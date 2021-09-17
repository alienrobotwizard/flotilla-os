package engines

import (
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"net/http"
)

type Engine interface {
	Name() string
	Enqueue(run models.Run) error
	Terminate(run models.Run) error
	Execute(run models.Run) (models.Run, error)
	Poll(callback func(models.Run) (shouldAck bool, err error)) error
	GetLatest(run models.Run) (models.Run, error)
	UpdateMetrics(run models.Run) (models.Run, error)
	Logs(template models.Template, run models.Run, lastSeen *string) (string, *string, error)
	LogsText(template models.Template, run models.Run, w http.ResponseWriter) error
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
