package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Engine struct {
	qm        *QueueManager
	queueName string
	logger    *log.Logger
	kClient   *kubernetes.Clientset
	adapter   *Adapter
}

var (
	EngineName    = "engine.kubernetes"
	amqpQueue     = "engine.kubernetes.amqp_queue"
	amqpDSNKey    = "engine.kubernetes.amqp_dsn"
	imagePullKey  = "engine.kubernetes.image_pull_secrets"
	configPathKey = "engine.kubernetes.kubeconf"
)

func newKubeClient(configPath string) (*kubernetes.Clientset, error) {
	c, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(c)
}

func NewEngine(conf *config.Config) (engines.Engine, error) {
	logger := log.New(os.Stdout, "Kubernetes Execution Engine: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Printf("Initializing kubernetes execution engine\n")

	if !conf.IsSet(amqpDSNKey) {
		return nil, fmt.Errorf("%s must be set in config", amqpDSNKey)
	}

	if !conf.IsSet(amqpQueue) {
		return nil, fmt.Errorf("%s must be set in config", amqpQueue)
	}

	kconf := os.ExpandEnv("$HOME/.kube/config")
	if conf.IsSet(configPathKey) {
		kconf = conf.GetString(configPathKey)
	}

	kClient, err := newKubeClient(kconf)
	if err != nil {
		return nil, err
	}

	qm, err := NewQueueManager(conf.GetString(amqpDSNKey))
	if err != nil {
		return nil, err
	}

	return &Engine{
		qm:        qm,
		queueName: conf.GetString(amqpQueue),
		logger:    logger,
		kClient:   kClient,
		adapter:   NewAdapter(conf, kClient),
	}, nil
}

func (e *Engine) Name() string {
	return "kubernetes"
}

func (e *Engine) Close() error {
	return e.qm.Close()
}

func (e *Engine) Enqueue(ctx context.Context, run models.Run) error {
	return e.qm.WithChannel(func(channel *amqp.Channel) error {
		queue, err := channel.QueueDeclare(e.queueName, true, false, false, false, nil)
		if err != nil {
			return err
		}

		msg, err := json.Marshal(run)
		if err != nil {
			return err
		}

		if err = channel.Publish(
			"", queue.Name, false, false, amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         msg,
			},
		); err != nil {
			return err
		}
		return nil
	})
}

func (e *Engine) Terminate(ctx context.Context, run models.Run) error {
	opts, err := e.adapter.GetRunOpts(run)
	if err != nil {
		return err
	}

	patch := []struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value int32  `json:"value"`
	}{{
		Op:    "replace",
		Path:  "/spec/parallelism",
		Value: 0,
	}}

	patchData, err := json.Marshal(patch)

	_, err = e.kClient.BatchV1().Jobs(opts.GetNamespace()).Patch(
		ctx, run.RunID, types.JSONPatchType, patchData, v1.PatchOptions{})
	return err
}

func (e *Engine) Execute(ctx context.Context, run models.Run) (models.Run, error) {
	job, err := e.adapter.RunToJob(run)
	if err != nil {
		return run, err
	}

	jobClient := e.kClient.BatchV1().Jobs(job.Namespace)
	launched, err := jobClient.Create(ctx, &job, v1.CreateOptions{})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return run, err
		}

		// Job spec is invalid, don't retry.
		if strings.Contains(strings.ToLower(err.Error()), "is invalid") {
			exitReason := err.Error()
			run.ExitReason = &exitReason
			return run, err
		}
		return run, errors.Wrap(exceptions.ErrRetryable, err.Error())
	}

	pod, err := e.getJobPod(ctx, job.Namespace, job.Name)
	if err != nil {
		return run, err
	}
	return e.adapter.JobToRun(launched, run, &pod)
}

func (e *Engine) Poll(ctx context.Context, callback func(models.Run) (shouldAck bool, err error)) error {
	return e.qm.WithChannel(func(channel *amqp.Channel) error {
		queue, err := channel.QueueDeclare(e.queueName, true, false, false, false, nil)
		if err != nil {
			return err
		}

		msg, ok, err := channel.Get(queue.Name, false)
		if err != nil {
			return err
		}

		if !ok {
			// No message
			return nil
		}

		var run models.Run
		if err = json.Unmarshal(msg.Body, &run); err != nil {
			return err
		}

		shouldAck, err := callback(run)
		if shouldAck {
			msg.Ack(false)
		}
		return err
	})
}

func (e *Engine) getJobPod(ctx context.Context, namespace string, runID string) (corev1.Pod, error) {
	var pod corev1.Pod
	listed, err := e.kClient.CoreV1().Pods(namespace).List(ctx, v1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", runID),
	})

	if err != nil {
		return pod, err
	}

	if listed != nil && listed.Items != nil && len(listed.Items) > 0 {
		pod = listed.Items[len(listed.Items)-1]
	}

	if len(pod.Name) == 0 {
		return pod, engines.ErrNotFound
	}
	return pod, nil
}

func (e *Engine) GetLatest(ctx context.Context, run models.Run) (models.Run, error) {
	opts, err := e.adapter.GetRunOpts(run)
	if err != nil {
		return run, err
	}

	job, err := e.kClient.BatchV1().Jobs(opts.GetNamespace()).Get(ctx, run.RunID, metav1.GetOptions{})
	if err != nil {
		return run, err
	}

	pod, err := e.getJobPod(ctx, job.Namespace, job.Name)
	if err != nil {
		return run, err
	}
	return e.adapter.JobToRun(job, run, &pod)
}

func (e *Engine) UpdateMetrics(ctx context.Context, run models.Run) (models.Run, error) {
	return models.Run{}, nil
}

func (e *Engine) Logs(
	ctx context.Context, template models.Template, run models.Run, lastSeen *string) (string, *string, error) {

	opts, err := e.adapter.GetRunOpts(run)
	if err != nil {
		return "", nil, err
	}

	pod, err := e.getJobPod(ctx, opts.GetNamespace(), run.RunID)
	if err != nil {
		return "", nil, err
	}

	since := time.Now().Format(time.RFC3339)
	logOpts := &corev1.PodLogOptions{
		Container: "main",
	}

	if lastSeen != nil {
		if t, err := time.Parse(time.RFC3339, *lastSeen); err != nil {
			// validation error?
		} else {
			logOpts.SinceTime = &metav1.Time{Time: t}
		}
	}

	req := e.kClient.CoreV1().Pods(opts.GetNamespace()).GetLogs(pod.Name, logOpts)
	if logs, err := req.Stream(ctx); err != nil {
		return "", nil, err
	} else {
		defer logs.Close()
		body, err := io.ReadAll(logs)
		if err != nil {
			return "", nil, err
		}
		return string(body), &since, nil
	}
}

func (e *Engine) LogsText(ctx context.Context, template models.Template, run models.Run, w http.ResponseWriter) error {
	return nil
}
