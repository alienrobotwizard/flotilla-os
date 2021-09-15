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

type StatusWorker struct {
	ctx          context.Context
	engine       engines.Engine
	pollInterval time.Duration
}

func (sw *StatusWorker) Initialize(c *config.Config, sm state.Manager, engine engines.Engine) error {
	pollInterval, err := GetWorkerPollInterval(c, models.StatusWorker)
	if err != nil {
		return err
	}
	sw.pollInterval = pollInterval
	return nil
}

func (sw *StatusWorker) Run(ctx context.Context, wg *sync.WaitGroup) error {
	sw.ctx = ctx
	go func() {
		defer wg.Done()
		fmt.Println("we start")
		for {
			select {
			case <-ctx.Done():
				fmt.Println("we die")
				return
			default:
				time.Sleep(sw.pollInterval)
			}
			fmt.Println("we run")
		}
	}()
	return nil
}
