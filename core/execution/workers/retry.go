package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/alienrobotwizard/flotilla-os/core/utils"
	"time"
)

type RetryWorker struct {
	sm           state.Manager
	engine       engines.Engine
	pollInterval time.Duration
}

func NewRetryWorker(c *config.Config, sm state.Manager, engine engines.Engine) (Worker, error) {
	pollInterval, err := GetWorkerPollInterval(c, models.RetryWorker)
	if err != nil {
		return nil, err
	}

	return &RetryWorker{
		sm:           sm,
		engine:       engine,
		pollInterval: pollInterval,
	}, nil
}

func (rw *RetryWorker) Run(ctx context.Context) error {
	go func(ctx context.Context) {
		t := time.NewTicker(rw.pollInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				rw.runOnce(ctx)
			}
		}
	}(ctx)
	return nil
}

func (rw *RetryWorker) runOnce(ctx context.Context) {
	// List runs in the StatusNeedsRetry state and requeue them
	engineFltr := state.EnginesList([]string{rw.engine.Name()})
	runList, err := rw.sm.ListRuns(ctx, &state.ListRunsArgs{
		ListArgs: state.ListArgs{
			Limit:  utils.IntP(25),
			Offset: utils.IntP(0),
			SortBy: utils.StringP("started_at"),
			Order:  utils.StringP("asc"),
			Filters: map[string][]string{
				"status": {
					string(models.StatusNeedsRetry),
				},
			},
		},
		Engines: &engineFltr,
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, run := range runList.Runs {
		if _, err = rw.sm.UpdateRun(ctx, run.RunID, models.Run{Status: models.StatusQueued}); err != nil {
			fmt.Println(err)
			return
		}

		if err = rw.engine.Enqueue(run); err != nil {
			fmt.Println(err)
			return
		}
	}
	return
}
