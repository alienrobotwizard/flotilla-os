package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/exceptions"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"log"
	"os"
	"time"
)

type SubmitWorker struct {
	sm           state.Manager
	engine       engines.Engine
	pollInterval time.Duration
	logger       *log.Logger
}

func NewSubmitWorker(c *config.Config, sm state.Manager, engine engines.Engine) (Worker, error) {
	pollInterval, err := GetWorkerPollInterval(c, models.SubmitWorker)
	if err != nil {
		return nil, err
	}
	return &SubmitWorker{
		sm:           sm,
		engine:       engine,
		pollInterval: pollInterval,
		logger:       log.New(os.Stderr, fmt.Sprintf("%s SubmitWorker: ", engine.Name()), log.LstdFlags),
	}, nil
}

func (sw *SubmitWorker) Run(ctx context.Context) error {
	sw.logger.Printf("Starting with poll interval [%s]\n", sw.pollInterval)
	go func(ctx context.Context) {
		t := time.NewTicker(sw.pollInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sw.runOnce(ctx)
			}
		}
	}(ctx)
	return nil
}

func (sw *SubmitWorker) runOnce(ctx context.Context) {
	err := sw.engine.Poll(ctx, func(run models.Run) (shouldAck bool, err error) {
		sw.logger.Printf("Processing run: [%s]\n", run.RunID)
		run, err = sw.sm.GetRun(ctx, run.RunID)
		if err != nil {
			return true, err
		}

		if run.Status == models.StatusQueued {
			_, err = sw.sm.GetTemplate(ctx, &state.GetTemplateArgs{TemplateID: run.TemplateID})

			if err != nil {
				return true, err
			}
			launched, err := sw.engine.Execute(ctx, run)
			if err != nil {
				if !errors.Is(err, exceptions.ErrRetryable) {
					launched.Status = models.StatusStopped
					launched.RunExceptions = &models.RunExceptions{err.Error()}
					return true, nil
				} else {
					return false, nil
				}
			}
			if _, err := sw.sm.UpdateRun(ctx, launched.RunID, launched); err != nil {
				return true, err
			}
			return true, nil
		} else if run.Status == models.StatusStopped {
			return true, nil
		}

		sw.logger.Printf("Received run: [%s] with non-queued status [%s], not-acking\n", run.RunID, run.Status)
		return false, nil
	})

	if err != nil && !errors.Is(err, engines.ErrNoRuns) {
		sw.logger.Printf("[ERROR]: %v\n", err)
	}
}
