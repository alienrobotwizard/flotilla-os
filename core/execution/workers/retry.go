package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"sync"
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

func (sw *RetryWorker) Run(ctx context.Context, wg *sync.WaitGroup) error {
	go func(ctx context.Context) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				fmt.Println("we die")
				return
			default:
				time.Sleep(sw.pollInterval)
			}
		}
	}(ctx)
	return nil
}
