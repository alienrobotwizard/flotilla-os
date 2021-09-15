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
	"sync"
	"time"
)

type SubmitWorker struct {
	ctx          context.Context
	sm           state.Manager
	engine       engines.Engine
	pollInterval time.Duration
}

func (sw *SubmitWorker) Initialize(c *config.Config, sm state.Manager, engine engines.Engine) error {
	pollInterval, err := GetWorkerPollInterval(c, models.SubmitWorker)
	if err != nil {
		return err
	}
	sw.sm = sm
	sw.engine = engine
	sw.pollInterval = pollInterval
	return nil
}

func (sw *SubmitWorker) Run(ctx context.Context, wg *sync.WaitGroup) error {
	sw.ctx = ctx
	go func() {
		t := time.NewTicker(sw.pollInterval)
		defer wg.Done()
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sw.runOnce()
			}
		}
	}()
	return nil
}

func (sw *SubmitWorker) runOnce() {
	err := sw.engine.Poll(func(run models.Run) (shouldAck bool, err error) {
		run, err = sw.sm.GetRun(run.RunID)
		if err != nil {
			return true, err
		}

		if run.Status == models.StatusQueued {
			_, err = sw.sm.GetTemplate(&state.GetTemplateArgs{TemplateID: run.TemplateID})

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
			if _, err := sw.sm.UpdateRun(launched.RunID, launched); err != nil {
				return true, err
			}
			return true, nil
		}
		return false, nil
	})
	fmt.Println(err)
}
