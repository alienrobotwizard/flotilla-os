package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"log"
	"os"
	"sync"
	"time"
)

type WorkerManager struct {
	sm           state.Manager
	conf         *config.Config
	engines      engines.Engines
	pollInterval time.Duration
	workers      map[string]engineWorkers
	logger       *log.Logger
}

//
// engineWorkers managers all the worker processes associated with a given engine
//
type engineWorkers struct {
	conf         *config.Config
	engine       engines.Engine
	ctx          context.Context
	stateManager state.Manager
	workers      map[models.WorkerType][]workerContext
	logger       *log.Logger
}

func newEngineWorkers(
	conf *config.Config, ctx context.Context, stateManager state.Manager, engine engines.Engine) engineWorkers {
	return engineWorkers{
		conf:         conf,
		ctx:          ctx,
		engine:       engine,
		stateManager: stateManager,
		workers:      make(map[models.WorkerType][]workerContext),
		logger:       log.New(os.Stderr, fmt.Sprintf("EngineWorkers[%s]: ", engine.Name()), log.LstdFlags),
	}
}

//
// addWorker will add a worker of a specific type to the pool
// onStop will be called for this specific worker process when either:
// a. the worker is removed naturally via the removeWorker function or
// b. the worker is stopped when this engineWorkers stop function is called
//
func (e *engineWorkers) addWorker(workerType models.WorkerType, onStop func()) error {
	toAdd, err := newWorker(e.conf, e.stateManager, e.engine, workerType)
	if err != nil {
		return err
	}

	childContext, cancel := context.WithCancel(e.ctx)
	cancelAndRemove := func() {
		e.logger.Printf("Removing worker of type [%s]\n", workerType)
		cancel()
		onStop()
	}

	if err = toAdd.Run(childContext); err != nil {
		cancelAndRemove()
		return err
	}

	e.workers[workerType] = append(e.workers[workerType], workerContext{
		ctx:    childContext,
		cancel: cancelAndRemove,
	})
	return nil
}

func (e *engineWorkers) stop() {
	e.logger.Println("Stopping all workers")
	for _, workers := range e.workers {
		for _, worker := range workers {
			worker.cancel()
		}
	}
}

func (e *engineWorkers) count(workerType models.WorkerType) int {
	return len(e.workers[workerType])
}

func (e *engineWorkers) removeWorker(workerType models.WorkerType) {
	if workers, ok := e.workers[workerType]; ok && len(workers) > 0 {
		toRemove := workers[len(workers)-1]
		toRemove.cancel()
		e.workers[workerType] = e.workers[workerType][:len(workers)-1]
	}
}

func newWorker(
	conf *config.Config, stateManager state.Manager,
	engine engines.Engine, workerType models.WorkerType) (worker Worker, err error) {

	switch workerType {
	case "retry":
		worker, err = NewRetryWorker(conf, stateManager, engine)
	case "status":
		worker, err = NewStatusWorker(conf, stateManager, engine)
	case "submit":
		worker, err = NewSubmitWorker(conf, stateManager, engine)
	default:
		err = fmt.Errorf("unimplemented worker type")
	}

	return
}

type workerContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewManager(c *config.Config, sm state.Manager, engines engines.Engines) (*WorkerManager, error) {
	pollInterval, err := GetWorkerPollInterval(c, models.ManagerWorker)
	if err != nil {
		return nil, err
	}

	wm := WorkerManager{
		sm:           sm,
		conf:         c,
		engines:      engines,
		pollInterval: pollInterval,
		workers:      make(map[string]engineWorkers),
		logger:       log.New(os.Stderr, "WorkerManager: ", log.LstdFlags),
	}

	return &wm, nil
}

//
// Start will start the WorkerManager process in a goroutine which will manage
// all worker types for all execution engines
//
func (wm *WorkerManager) Start(ctx context.Context) {
	wm.logger.Printf("Starting with poll interval [%s]\n", wm.pollInterval)
	go func(ctx context.Context) {
		// Use a WaitGroup to manage workers and graceful shutdown
		wg := &sync.WaitGroup{}
		defer wg.Wait()
		for engineName, engine := range wm.engines {
			wm.logger.Printf("Creating new workers for engine [%s]\n", engineName)
			wm.workers[engineName] = newEngineWorkers(wm.conf, ctx, wm.sm, engine)
		}
		wm.runOnce(ctx, wg)

		t := time.NewTicker(wm.pollInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				wm.Stop()
				return
			case <-t.C:
				wm.runOnce(ctx, wg)
			}
		}
	}(ctx)
}

func (wm *WorkerManager) Stop() {
	for _, workers := range wm.workers {
		workers.stop()
	}
}

func (wm *WorkerManager) runOnce(ctx context.Context, wg *sync.WaitGroup) {
	if err := wm.updateWorkers(ctx, wg); err != nil {
		wm.logger.Printf("[ERROR]: %v\n", err)
	}
}

func (wm *WorkerManager) updateWorkers(ctx context.Context, wg *sync.WaitGroup) error {
	for engineName := range wm.engines {
		workersForEngine := wm.workers[engineName]
		if workerList, err := wm.sm.ListWorkers(ctx, engineName); err != nil {
			return err
		} else {
			for _, workerInfo := range workerList.Workers {
				workerType := models.WorkerType(workerInfo.WorkerType)
				existingCount := workersForEngine.count(workerType)
				desiredCount := workerInfo.CountPerInstance
				if existingCount > desiredCount {
					for i := 0; i < existingCount-desiredCount; i++ {
						workersForEngine.removeWorker(workerType)
					}
				} else if existingCount < desiredCount {
					for i := 0; i < desiredCount-existingCount; i++ {
						// Add to WaitGroup when worker is added, taking care
						// to remove it (call Done) when the worker is stopped or removed
						wg.Add(1)
						if err := workersForEngine.addWorker(workerType, func() {
							wg.Done()
						}); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func GetWorkerPollInterval(c *config.Config, workerType models.WorkerType) (time.Duration, error) {
	var interval time.Duration
	pollIntervalString := c.GetString(fmt.Sprintf("worker.%s_interval", workerType))
	if len(pollIntervalString) == 0 {
		return interval, errors.Errorf("worker type: [%s] needs worker.%s_interval set", workerType, workerType)
	}
	return time.ParseDuration(pollIntervalString)
}
