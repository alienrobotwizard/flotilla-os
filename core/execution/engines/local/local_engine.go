package local

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"log"
	"net/http"
	"os"
)

type Engine struct {
	logger *log.Logger
	queue  chan models.Run
}

func (e *Engine) Initialize(ctx context.Context, conf *config.Config, manager state.Manager) error {
	e.logger = log.New(os.Stdout, "Local Execution Engine: ", log.Ldate|log.Ltime|log.Lshortfile)
	e.logger.Printf("Initializing local execution engine\n")

	e.queue = make(chan models.Run)
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			close(e.queue)
		}
	}(ctx)
	return nil
}

func (e *Engine) Enqueue(run models.Run) error {
	e.logger.Printf("Enqueuing run with id: [%s]\n", run.RunID)
	e.queue <- run
	return nil
}

func (e *Engine) Terminate(run models.Run) error {
	fmt.Printf("Terminating run with id: [%s]\n", run.RunID)
	return nil
}

func (e *Engine) Execute(run models.Run) (models.Run, error) {
	// TODO - to run locally means to run with assumption docker-server is running locally?
	e.logger.Printf("Executing run with id: [%s]\n", run.RunID)
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

func (e *Engine) Logs(template models.Template, run models.Run, lastSeen *string) (string, *string, error) {
	e.logger.Println("getting logs")
	return "", nil, nil
}

func (e *Engine) LogsText(template models.Template, run models.Run, w http.ResponseWriter) error {
	e.logger.Println("getting logs text")
	return nil
}
