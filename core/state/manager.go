package state

import (
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
)

type EnginesList []string
type EnvFilters map[string]string

var (
	DefaultLimit  = 500
	DefaultOffset = 0
)

type ListArgs struct {
	Limit   *int
	Offset  *int
	SortBy  *string
	Order   *string
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
	// TODO - rather than cleanup, use appContext
	Initialize(c *config.Config) error
	Cleanup() error

	GetRun(runID string) (models.Run, error)
	ListRuns(args *ListRunsArgs) (models.RunList, error)
	CreateRun(r models.Run) (models.Run, error)
	UpdateRun(runID string, updates models.Run) (models.Run, error)

	GetTemplate(args *GetTemplateArgs) (models.Template, error)
	ListTemplates(args *ListArgs) (models.TemplateList, error)
	CreateTemplate(t models.Template) (models.Template, error)

	ListWorkers(engine string) (models.WorkersList, error)
	BatchUpdateWorkers(updates []models.Worker) (models.WorkersList, error)
	GetWorker(workerType models.WorkerType, engine string) (models.Worker, error)
	UpdateWorker(workerType models.WorkerType, updates models.Worker) (models.Worker, error)
}
