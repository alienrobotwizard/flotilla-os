package engines

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"net/http"
)

type Engine interface {
	Name() string
	Enqueue(ctx context.Context, run models.Run) error
	Terminate(ctx context.Context, run models.Run) error
	Execute(ctx context.Context, run models.Run) (models.Run, error)
	Poll(ctx context.Context, callback func(models.Run) (shouldAck bool, err error)) error
	GetLatest(ctx context.Context, run models.Run) (models.Run, error)
	UpdateMetrics(ctx context.Context, run models.Run) (models.Run, error)
	Logs(ctx context.Context, template models.Template, run models.Run, lastSeen *string) (string, *string, error)
	LogsText(ctx context.Context, template models.Template, run models.Run, w http.ResponseWriter) error
	Close() error
}

type Engines map[string]Engine

func (e Engines) Get(name string) (Engine, bool) {
	if e != nil {
		eng, ok := e[name]
		return eng, ok
	}
	return nil, false
}

var (
	ErrQueueClosed = errors.New("queue channel closed")
	ErrNoRuns      = errors.New("no runs")
	ErrNotFound    = errors.New("resource not found; either cleaned up or incorrectly specified")
)
