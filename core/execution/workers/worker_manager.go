package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type WorkerManager struct {
	sm      state.Manager
	conf    *config.Config
	engines map[string]engines.Engine
	workers map[string][]workerContext
}

type workerContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewManager(c *config.Config, sm state.Manager, engines map[string]engines.Engine) (*WorkerManager, error) {
	wm := WorkerManager{
		sm:      sm,
		conf:    c,
		engines: engines,
		workers: make(map[string][]workerContext),
	}
	return &wm, nil
}

func (wm *WorkerManager) Start(ctx context.Context) (*sync.WaitGroup, error) {
	wg := &sync.WaitGroup{}
	for engineName, engine := range wm.engines {
		if workerList, err := wm.sm.ListWorkers(engineName); err != nil {
			return nil, err
		} else {
			wg.Add(int(workerList.Total))
			for _, worker := range workerList.Workers {
				wm.workers[worker.WorkerType] = make([]workerContext, worker.CountPerInstance)
				for i := 0; i < worker.CountPerInstance; i++ {
					if wctx, cancelFunc, err := wm.newWorker(ctx, wg, worker.WorkerType, engine); err != nil {
						return nil, err
					} else {
						wm.workers[worker.WorkerType][i] = workerContext{ctx: wctx, cancel: cancelFunc}
					}
				}
			}
		}
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	}()
	return wg, nil
}

func (wm *WorkerManager) newWorker(
	ctx context.Context, wg *sync.WaitGroup, workerType string, engine engines.Engine) (context.Context, context.CancelFunc, error) {
	var worker Worker
	switch workerType {
	case "retry":
		worker = &RetryWorker{}
	case "status":
		worker = &StatusWorker{}
	case "submit":
		worker = &SubmitWorker{}
	}

	if err := worker.Initialize(wm.conf, wm.sm, engine); err != nil {
		return nil, nil, err
	}

	child, f := context.WithCancel(ctx)
	return child, f, worker.Run(child, wg)
}

func GetWorkerPollInterval(c *config.Config, workerType models.WorkerType) (time.Duration, error) {
	var interval time.Duration
	pollIntervalString := c.GetString(fmt.Sprintf("worker.%s_interval", workerType))
	if len(pollIntervalString) == 0 {
		return interval, errors.Errorf("worker type: [%s] needs worker.%s_interval set", workerType, workerType)
	}
	return time.ParseDuration(pollIntervalString)
}
