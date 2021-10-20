package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/alienrobotwizard/flotilla-os/core/utils"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

type RunOpts struct {
	Secrets            []string          `json:"secrets"`
	NoEntrypoint       bool              `json:"no_entrypoint"`
	NodeSelectors      map[string]string `json:"node_selectors"`
	Namespace          *string           `json:"namespace,omitempty"`
	ServiceAccountName *string           `json:"service_account_name,omitempty"`
}

func (o RunOpts) GetNamespace() string {
	if o.Namespace != nil {
		return *o.Namespace
	}
	return "default"
}

type Adapter struct {
	imagePullSecrets string
	client           *kubernetes.Clientset
}

func NewAdapter(c *config.Config, client *kubernetes.Clientset) *Adapter {
	return &Adapter{client: client, imagePullSecrets: c.GetString(imagePullKey)}
}

func (a *Adapter) JobToRun(job *batchv1.Job, existing models.Run, pod *corev1.Pod) (models.Run, error) {
	updated := existing
	if pod != nil && pod.Spec.Containers != nil && len(pod.Spec.Containers) > 0 {
		// TODO - we would ideally like multiple containers per job for sidecar behavior
		main := pod.Spec.Containers[len(pod.Spec.Containers)-1]
		cpu := main.Resources.Requests.Cpu().ScaledValue(resource.Milli)
		cpuLimit := main.Resources.Limits.Memory().ScaledValue(resource.Milli)
		mem := main.Resources.Requests.Memory().ScaledValue(resource.Mega)
		memLimit := main.Resources.Limits.Memory().ScaledValue(resource.Mega)
		updated.Cpu = &cpu
		updated.Memory = &mem
		updated.CpuLimit = &cpuLimit
		updated.MemoryLimit = &memLimit

		if len(pod.Spec.NodeName) > 0 {
			updated.InstanceDNSName = pod.Spec.NodeName
		}
	}

	if job.Status.Active == 1 && job.Status.CompletionTime == nil {
		updated.Status = models.StatusRunning
	} else if job.Status.Succeeded == 1 {
		if pod != nil {
			if pod.Status.Phase == corev1.PodSucceeded {
				var exitCode int64 = 0
				var exitReason = fmt.Sprintf("Pod %s Exited Successfully", pod.Name)
				updated.ExitReason = &exitReason
				updated.Status = models.StatusStopped
				updated.ExitCode = &exitCode
			}
		} else {
			var exitCode int64 = 0
			updated.Status = models.StatusStopped
			updated.ExitCode = &exitCode
		}
	} else if job.Status.Failed == 1 {
		var exitCode int64 = 1
		updated.Status = models.StatusStopped
		if pod != nil {
			if pod.Status.ContainerStatuses != nil && len(pod.Status.ContainerStatuses) > 0 {
				containerStatus := pod.Status.ContainerStatuses[len(pod.Status.ContainerStatuses)-1]
				if containerStatus.State.Terminated != nil {
					updated.ExitReason = &containerStatus.State.Terminated.Reason
					exitCode = int64(containerStatus.State.Terminated.ExitCode)
				}
			}
		}
		updated.ExitCode = &exitCode
	}

	if job != nil && job.Status.StartTime != nil {
		updated.StartedAt = &job.Status.StartTime.Time
	}

	if updated.Status == models.StatusStopped {
		if job != nil && job.Status.CompletionTime != nil {
			updated.FinishedAt = &job.Status.CompletionTime.Time
		} else {
			finishedAt := time.Now()
			updated.FinishedAt = &finishedAt
		}
	}

	return updated, nil
}

func (a *Adapter) RunToJob(run models.Run) (batchv1.Job, error) {
	var job batchv1.Job

	opts, err := a.GetRunOpts(run)
	if err != nil {
		return job, err
	}

	cmd := ""
	if run.Command != nil && len(*run.Command) > 0 {
		cmd = *run.Command
	}

	cmdSlice := []string{cmd}
	if !opts.NoEntrypoint {
		cmdSlice = append([]string{"bash", "-l", "-cex"}, cmdSlice...)
	}

	var env []corev1.EnvVar
	if run.Env != nil {
		env = a.handleEnv(*run.Env)
	}

	secrets, err := a.handleSecrets(opts.GetNamespace(), opts.Secrets)
	if err != nil {
		return job, err
	}

	env = append(env, secrets...)

	container := corev1.Container{
		Name:            "main",
		Image:           run.Image,
		ImagePullPolicy: corev1.PullAlways,
		Env:             env,
		//VolumeMounts: []corev1.VolumeMount{
		//	{
		//		Name:      "flotilla-secrets-vol",
		//		ReadOnly:  true,
		//		MountPath: "/flotilla/secrets",
		//	},
		//},
		Resources: a.handleResourceRequirements(run),
	}

	if opts.NoEntrypoint {
		container.Command = cmdSlice
	} else {
		container.Args = cmdSlice
	}

	annotations := map[string]string{"cluster-autoscaler.kubernetes.io/safe-to-evict": "false"}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		ImagePullSecrets: []corev1.LocalObjectReference{
			{Name: a.imagePullSecrets},
		},

		Containers:   []corev1.Container{container},
		NodeSelector: opts.NodeSelectors,
	}

	if opts.ServiceAccountName != nil {
		podSpec.ServiceAccountName = *opts.ServiceAccountName
	}

	spec := batchv1.JobSpec{
		Parallelism:             utils.Int32P(1),
		BackoffLimit:            utils.Int32P(0),
		TTLSecondsAfterFinished: utils.Int32P(60),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
			},
			Spec: podSpec,
		},
	}

	return batchv1.Job{
		Spec: spec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.RunID,
			Namespace: opts.GetNamespace(),
		},
	}, nil
}

func (a *Adapter) handleSecrets(namespace string, secrets []string) ([]corev1.EnvVar, error) {
	// TODO - properly use parent context
	res, err := a.client.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "",
		FieldSelector: "",
	})
	if err != nil {
		return nil, err
	}

	toSet := make(map[string]bool)
	for _, s := range secrets {
		toSet[s] = true
	}

	var result []corev1.EnvVar
	for _, s := range res.Items {
		if toSet[s.Name] {
			for k, _ := range s.Data {
				evar := corev1.EnvVar{
					Name: k,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: s.Name},
							Key:                  k,
						},
					},
				}
				result = append(result, evar)
			}
		}
	}

	return result, nil
}

func (a *Adapter) handleResourceRequirements(run models.Run) corev1.ResourceRequirements {
	reqs := corev1.ResourceList{}
	lims := corev1.ResourceList{}

	if run.Memory != nil && *run.Memory > 0 {
		reqs[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dM", run.Memory))
	}

	if run.Cpu != nil && *run.Cpu > 0 {
		reqs[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", run.Cpu))
	}

	if run.MemoryLimit != nil && *run.MemoryLimit > 0 {
		lims[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dM", run.MemoryLimit))
	}

	if run.CpuLimit != nil && *run.CpuLimit > 0 {
		lims[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", run.CpuLimit))
	}

	return corev1.ResourceRequirements{
		Requests: reqs,
		Limits:   lims,
	}
}

func (a *Adapter) handleEnv(el models.EnvList) []corev1.EnvVar {
	env := make([]corev1.EnvVar, len(el))
	for i, e := range el {
		env[i] = corev1.EnvVar{Name: e.Name, Value: e.Value}
	}
	return env
}

func (a *Adapter) GetRunOpts(run models.Run) (RunOpts, error) {
	var (
		err  error
		opts RunOpts
	)

	if run.EngineArgs != nil {
		err = json.Unmarshal(*run.EngineArgs, &opts)
	}
	return opts, err
}
