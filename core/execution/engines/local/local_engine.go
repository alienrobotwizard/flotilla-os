package local

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"time"
)

type Engine struct {
	logger *log.Logger
	queue  chan models.Run
	docker DockerClient
}

var (
	ErrQueueCapacity = errors.New("job queue is full")
)

func NewLocalEngine(ctx context.Context, conf *config.Config) (engines.Engine, error) {
	logger := log.New(os.Stdout, "Local Execution Engine: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Printf("Initializing local execution engine\n")

	dc, err := NewDockerClient(conf)
	if err != nil {
		return nil, err
	}

	queue := make(chan models.Run, 100)
	go func(ctx context.Context, queue chan models.Run) {
		select {
		case <-ctx.Done():
			close(queue)
		}
	}(ctx, queue)

	return &Engine{
		docker: dc,
		logger: logger,
		queue:  queue,
	}, nil
}

func (e *Engine) Name() string {
	return "local"
}

func (e *Engine) Enqueue(run models.Run) error {
	e.logger.Printf("Enqueuing run with id: [%s]\n", run.RunID)
	if len(e.queue) < 100 {
		e.queue <- run
	} else {
		return ErrQueueCapacity
	}
	return nil
}

func (e *Engine) Terminate(run models.Run) error {
	return e.docker.Terminate(context.Background(), run)
}

func (e *Engine) Execute(run models.Run) (models.Run, error) {
	_, err := e.docker.Execute(context.Background(), run)
	if err != nil {
		return run, err
	}
	run.Status = models.StatusPending
	return run, nil
}

func (e *Engine) Poll(callback func(models.Run) (shouldAck bool, err error)) error {
	var (
		ok  bool
		run models.Run
	)

	select {
	case run, ok = <-e.queue:
		if !ok {
			return engines.ErrQueueClosed
		}
		shouldAck, err := callback(run)
		if !shouldAck {
			e.queue <- run
		}
		return err
	default:
		return engines.ErrNoRuns
	}
}

func (e *Engine) GetLatest(run models.Run) (models.Run, error) {

	info, err := e.docker.Info(context.Background(), run)
	if err != nil {
		return run, err
	}

	state := info.State
	if state != nil {
		switch state.Status {
		case "created":
			run.Status = models.StatusPending
			run.InstanceID = info.Node.ID
			run.InstanceDNSName = info.Node.Addr
		case "running":
			run.Status = models.StatusRunning
		default:
			run.Status = models.StatusStopped
			exitCode := int64(state.ExitCode)
			run.ExitCode = &exitCode
			if len(state.Error) > 0 {
				run.ExitReason = &state.Error
			}
		}

		finished, _ := time.Parse(time.RFC3339Nano, state.FinishedAt)
		started, _ := time.Parse(time.RFC3339Nano, state.StartedAt)
		if !started.IsZero() {
			run.StartedAt = &started
		}

		if !finished.IsZero() {
			run.FinishedAt = &finished
		}
	}

	return run, nil
}

func (e *Engine) UpdateMetrics(run models.Run) (models.Run, error) {
	// Not supported yet
	return run, nil
}

func (e *Engine) Logs(template models.Template, run models.Run, lastSeen *string) (string, *string, error) {
	return e.docker.Logs(context.Background(), run, lastSeen)
}

func (e *Engine) LogsText(template models.Template, run models.Run, w http.ResponseWriter) error {
	e.logger.Println("getting logs text")
	return nil
}
