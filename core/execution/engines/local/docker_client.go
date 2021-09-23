package local

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"io"
	"os"
	"time"
)

type DockerClient interface {
	Execute(ctx context.Context, run models.Run) (containerID string, err error)
	Terminate(ctx context.Context, run models.Run) error
	Logs(ctx context.Context, run models.Run, lastSeen *string) (string, *string, error)
	Info(ctx context.Context, run models.Run) (types.ContainerJSON, error)
}

type dockerClient struct {
	cli client.APIClient
}

func NewDockerClient(c *config.Config) (DockerClient, error) {
	// TODO - config should provide multiple paths for docker host and auth
	if cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
		return nil, err
	} else {
		return &dockerClient{
			cli: cli,
		}, nil
	}
}

func (dc *dockerClient) Execute(ctx context.Context, run models.Run) (containerID string, err error) {
	reader, err := dc.cli.ImagePull(ctx, run.Image, types.ImagePullOptions{})
	if err != nil {
		return
	}
	io.Copy(os.Stdout, reader)

	var env []string
	if run.Env != nil {
		env := make([]string, len(*run.Env))
		for i, ev := range *run.Env {
			env[i] = fmt.Sprintf("%s=%s", ev.Name, ev.Value)
		}
	}

	var resources container.Resources
	if run.Cpu != nil {
		resources.CPUShares = *run.Cpu
	}

	if run.Memory != nil {
		resources.MemoryReservation = (*run.Memory) * 1000000
	}

	resp, err := dc.cli.ContainerCreate(ctx, &container.Config{
		Image: run.Image,
		Cmd:   []string{"bash", "-l", "-cex", *run.Command},
		Env:   env,
	}, &container.HostConfig{
		Resources: resources,
	}, nil, nil, run.RunID)
	if err != nil {
		return
	}
	return resp.ID, dc.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
}

func (dc *dockerClient) getContainerID(ctx context.Context, run models.Run) (string, error) {
	containers, err := dc.cli.ContainerList(ctx, types.ContainerListOptions{
		Limit:   1,
		Filters: filters.NewArgs(filters.Arg("name", run.RunID)),
	})
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", engines.ErrNotFound
	}
	return containers[0].ID, nil
}

func (dc *dockerClient) Terminate(ctx context.Context, run models.Run) error {
	if containerID, err := dc.getContainerID(ctx, run); err != nil {
		return err
	} else {
		return dc.cli.ContainerStop(ctx, containerID, nil)
	}
}

func (dc *dockerClient) Info(ctx context.Context, run models.Run) (types.ContainerJSON, error) {
	if containerID, err := dc.getContainerID(ctx, run); err != nil {
		return types.ContainerJSON{}, err
	} else {
		return dc.cli.ContainerInspect(ctx, containerID)
	}
}

func (dc *dockerClient) Logs(ctx context.Context, run models.Run, lastSeen *string) (string, *string, error) {
	containerID, err := dc.getContainerID(ctx, run)
	if err != nil {
		return "", nil, err
	}

	logOpts := types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true}
	if lastSeen != nil {
		logOpts.Since = *lastSeen
	}

	out, err := dc.cli.ContainerLogs(ctx, containerID, logOpts)
	since := time.Now().Format(time.RFC3339)
	if err != nil {
		return "", nil, err
	}
	b, err := io.ReadAll(out)
	if err != nil {
		return "", nil, err
	}

	return string(b), &since, nil
}
