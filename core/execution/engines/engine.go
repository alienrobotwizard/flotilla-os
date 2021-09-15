package engines

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"github.com/alienrobotwizard/flotilla-os/core/state/models"
	"github.com/pkg/errors"
	"net/http"
)

type Engine interface {
	Initialize(ctx context.Context, conf *config.Config, manager state.Manager) error
	Enqueue(run models.Run) error
	Terminate(run models.Run) error
	Execute(run models.Run) (models.Run, error)
	Poll(callback func(models.Run) (shouldAck bool, err error)) error
	Logs(template models.Template, run models.Run, lastSeen *string) (string, *string, error)
	LogsText(template models.Template, run models.Run, w http.ResponseWriter) error
}

var (
	ErrQueueClosed = errors.New("queue channel closed")
	ErrNoRuns      = errors.New("no runs")
)
