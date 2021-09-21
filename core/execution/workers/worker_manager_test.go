package workers

import (
	"context"
	"fmt"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines/local"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

type mockManager struct {
	workers map[string]models.WorkersList
}

func TestWorkerManager_Start(t *testing.T) {
	// 1. Ensure that the proper number of listed workers are started
	// 2. Parent context completing means all workers also complete
	expectedWorkerCounts := map[models.WorkerType]int{
		models.StatusWorker: 2,
		models.RetryWorker:  1,
		models.SubmitWorker: 1,
	}

	expectedWorkers := []models.Worker{
		{WorkerType: string(models.StatusWorker), CountPerInstance: expectedWorkerCounts[models.StatusWorker]},
		{WorkerType: string(models.RetryWorker), CountPerInstance: expectedWorkerCounts[models.RetryWorker]},
		{WorkerType: string(models.SubmitWorker), CountPerInstance: expectedWorkerCounts[models.SubmitWorker]},
	}

	sm := &mockManager{workers: map[string]models.WorkersList{
		"local": {
			Total:   int64(len(expectedWorkers)),
			Workers: expectedWorkers,
		},
	}}

	conf, _ := config.NewConfig(nil)
	conf.Set(fmt.Sprintf("worker.%s_interval", models.ManagerWorker), "1s")
	conf.Set(fmt.Sprintf("worker.%s_interval", models.SubmitWorker), "1s")
	conf.Set(fmt.Sprintf("worker.%s_interval", models.RetryWorker), "2s")
	conf.Set(fmt.Sprintf("worker.%s_interval", models.StatusWorker), "10s")

	engine, _ := local.NewLocalEngine(context.Background(), conf)
	wm, _ := NewManager(conf, sm, map[string]engines.Engine{"local": engine})

	ctx, cancel := context.WithCancel(context.Background())
	wm.Start(ctx)

	localEngineWorkers := wm.workers["local"]

	for workerType, workerContexts := range localEngineWorkers.workers {
		expected, ok := expectedWorkerCounts[workerType]
		assert.True(t, ok, "expected %s worker type to be created", workerType)
		assert.Equal(t, expected, len(workerContexts))
		for _, wc := range workerContexts {
			assert.NoError(t, wc.ctx.Err(), "worker context should not be done")
		}
	}
	cancel()

	for _, workerContexts := range localEngineWorkers.workers {
		for _, wc := range workerContexts {
			assert.True(t, errors.Is(wc.ctx.Err(), context.Canceled))
		}
	}
}

func (mm *mockManager) GetRun(ctx context.Context, runID string) (models.Run, error) {
	return models.Run{}, nil
}

func (mm *mockManager) ListRuns(ctx context.Context, args *state.ListRunsArgs) (models.RunList, error) {
	return models.RunList{}, nil
}
func (mm *mockManager) CreateRun(ctx context.Context, r models.Run) (models.Run, error) {
	return models.Run{}, nil
}

func (mm *mockManager) UpdateRun(ctx context.Context, runID string, updates models.Run) (models.Run, error) {
	return models.Run{}, nil
}

func (mm *mockManager) GetTemplate(ctx context.Context, args *state.GetTemplateArgs) (models.Template, error) {
	return models.Template{}, nil
}

func (mm *mockManager) ListTemplates(ctx context.Context, args *state.ListArgs) (models.TemplateList, error) {
	return models.TemplateList{}, nil
}
func (mm *mockManager) CreateTemplate(ctx context.Context, t models.Template) (models.Template, error) {
	return models.Template{}, nil
}

func (mm *mockManager) ListWorkers(ctx context.Context, engine string) (models.WorkersList, error) {
	return mm.workers[engine], nil
}

func (mm *mockManager) BatchUpdateWorkers(ctx context.Context, updates []models.Worker) (models.WorkersList, error) {
	return models.WorkersList{}, nil
}

func (mm *mockManager) GetWorker(ctx context.Context, workerType models.WorkerType, engine string) (models.Worker, error) {
	return models.Worker{}, nil
}
func (mm *mockManager) UpdateWorker(ctc context.Context, workerType models.WorkerType, updates models.Worker) (models.Worker, error) {
	return models.Worker{}, nil
}
