package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/alienrobotwizard/flotilla-os/core/utils"
	"github.com/pkg/errors"
	"log"
	"os"
	"time"
)

type StatusWorker struct {
	sm           state.Manager
	engine       engines.Engine
	pollInterval time.Duration
	logger       *log.Logger
}

func NewStatusWorker(c *config.Config, sm state.Manager, engine engines.Engine) (Worker, error) {
	pollInterval, err := GetWorkerPollInterval(c, models.StatusWorker)
	if err != nil {
		return nil, err
	}

	return &StatusWorker{
		sm:           sm,
		engine:       engine,
		pollInterval: pollInterval,
		logger:       log.New(os.Stderr, fmt.Sprintf("%s StatusWorker: ", engine.Name()), log.LstdFlags),
	}, nil
}

func (sw *StatusWorker) Run(ctx context.Context) error {
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

func (sw *StatusWorker) runOnce(ctx context.Context) {
	engineFltr := state.EnginesList([]string{sw.engine.Name()})
	runs, err := sw.sm.ListRuns(ctx, &state.ListRunsArgs{
		ListArgs: state.ListArgs{
			Limit:  utils.IntP(1000),
			Offset: utils.IntP(0),
			SortBy: utils.StringP("started_at"),
			Order:  utils.StringP("asc"),
			Filters: map[string][]string{
				"queued_at_since": {time.Now().AddDate(0, 0, -30).Format(time.RFC3339)},
				"status": {
					string(models.StatusNeedsRetry), string(models.StatusRunning),
					string(models.StatusQueued), string(models.StatusPending),
				},
			},
		},
		Engines: &engineFltr,
	})

	if err != nil {
		sw.logger.Printf("[ERROR]: %v\n", err)
		return
	}

	if runs.Total > 0 {
		sw.logger.Printf("Got [%v] runs to update status for\n", runs.Total)
	}

	for _, run := range runs.Runs {
		updated, err := sw.engine.GetLatest(ctx, run)
		if err != nil {
			sw.logger.Printf("[ERROR]: Problem fetching run [%s] from engine\n%v\n", run.RunID, err)
			if !errors.Is(err, engines.ErrNotFound) {
				continue
			}
			if run.Status != models.StatusQueued {
				sw.sm.UpdateRun(
					ctx, run.RunID, models.Run{
						Status:     models.StatusStopped,
						ExitReason: utils.StringP("engine cannot find run"),
					})
			}
		}

		if updated.Status != run.Status {
			if updated.ExitCode != nil {
				go sw.cleanupRun(ctx, run.RunID)
			}

			_, err := sw.sm.UpdateRun(ctx, run.RunID, updated)
			if err != nil {
				sw.logger.Printf("[ERROR]: %v\n", err)
			}
		}
	}
}

func (sw *StatusWorker) cleanupRun(ctx context.Context, runID string) {
	// Wait a reasonable time for any external (outside of our control) processes to finish
	time.Sleep(120 * time.Second)
	if run, err := sw.sm.GetRun(ctx, runID); err == nil {
		_ = sw.engine.Terminate(ctx, run)
	}
}
