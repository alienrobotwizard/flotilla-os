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
	"time"
)

type SubmitWorker struct {
	sm           state.Manager
	engine       engines.Engine
	pollInterval time.Duration
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
	}, nil
}

func (sw *SubmitWorker) Run(ctx context.Context) error {
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
	err := sw.engine.Poll(func(run models.Run) (shouldAck bool, err error) {
		run, err = sw.sm.GetRun(ctx, run.RunID)
		if err != nil {
			return true, err
		}

		if run.Status == models.StatusQueued {
			_, err = sw.sm.GetTemplate(ctx, &state.GetTemplateArgs{TemplateID: run.TemplateID})

			if err != nil {
				return true, err
			}
			launched, err := sw.engine.Execute(run)
			if err != nil {
				if !errors.Is(err, exceptions.ErrRetryable) {
					launched.Status = models.StatusStopped
				} else {
					return false, nil
				}
			}
			if _, err := sw.sm.UpdateRun(ctx, launched.RunID, launched); err != nil {
				return true, err
			}
			return true, nil
		}
		return false, nil
	})
	fmt.Println(err)
}
