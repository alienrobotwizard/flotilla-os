package services

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
)

type WorkerService interface {
	List(ctx context.Context, engine string) (models.WorkersList, error)
	Get(ctx context.Context, workerType string, engine string) (models.Worker, error)
	Update(ctx context.Context, workerType string, updates models.Worker) (models.Worker, error)
	BatchUpdate(ctx context.Context, updates []models.Worker) (models.WorkersList, error)
}

type workerService struct {
	sm state.Manager
}

func NewWorkerService(c *config.Config, sm state.Manager) (WorkerService, error) {
	return &workerService{sm: sm}, nil
}

func (s *workerService) List(ctx context.Context, engine string) (models.WorkersList, error) {
	return s.sm.ListWorkers(ctx, engine)
}

func (s *workerService) Get(ctx context.Context, workerType string, engine string) (models.Worker, error) {
	return s.sm.GetWorker(ctx, models.WorkerType(workerType), engine)
}

func (s *workerService) Update(ctx context.Context, workerType string, updates models.Worker) (models.Worker, error) {
	return s.sm.UpdateWorker(ctx, models.WorkerType(workerType), updates)
}

func (s *workerService) BatchUpdate(ctx context.Context, updates []models.Worker) (models.WorkersList, error) {
	return s.sm.BatchUpdateWorkers(ctx, updates)
}
