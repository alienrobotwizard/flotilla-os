package state

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
)

type EnginesList []string
type EnvFilters map[string]string

var (
	DefaultLimit  = 500
	DefaultOffset = 0
)

type ListArgs struct {
	Limit   *int    `form:"limit"`
	Offset  *int    `form:"offset"`
	SortBy  *string `form:"sort_by"`
	Order   *string `form:"order"`
	Filters map[string][]string
}

func (args *ListArgs) AddFilter(key string, value string) {
	if args.Filters == nil {
		args.Filters = make(map[string][]string)
	}

	args.Filters[key] = append(args.Filters[key], value)
}

func (args *ListArgs) GetLimit() int {
	if args.Limit != nil {
		return *args.Limit
	}
	return DefaultLimit
}

func (args *ListArgs) GetOrder() string {
	if args.Order != nil {
		return *args.Order
	}
	return "asc"
}

func (args *ListArgs) GetOffset() int {
	if args.Offset != nil {
		return *args.Offset
	}
	return DefaultOffset
}

type ListRunsArgs struct {
	ListArgs
	EnvFilters *EnvFilters
	Engines    *EnginesList
}

type GetTemplateArgs struct {
	TemplateID      *string
	TemplateName    *string
	TemplateVersion *int64
}

//
// Manager interface for facilitating all interaction with databases
//
type Manager interface {
	GetRun(ctx context.Context, runID string) (models.Run, error)
	ListRuns(ctx context.Context, args *ListRunsArgs) (models.RunList, error)
	CreateRun(ctx context.Context, r models.Run) (models.Run, error)
	UpdateRun(ctx context.Context, runID string, updates models.Run) (models.Run, error)

	GetTemplate(ctx context.Context, args *GetTemplateArgs) (models.Template, error)
	ListTemplates(ctx context.Context, args *ListArgs) (models.TemplateList, error)
	CreateTemplate(ctx context.Context, t models.Template) (models.Template, error)

	ListWorkers(ctx context.Context, engine string) (models.WorkersList, error)
	BatchUpdateWorkers(ctx context.Context, updates []models.Worker) (models.WorkersList, error)
	GetWorker(ctx context.Context, workerType models.WorkerType, engine string) (models.Worker, error)
	UpdateWorker(ctc context.Context, workerType models.WorkerType, updates models.Worker) (models.Worker, error)
}
