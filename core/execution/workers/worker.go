package workers

import (
	"context"
	"github.com/alienrobotwizard/flotilla-os/core/config"
	"github.com/alienrobotwizard/flotilla-os/core/execution/engines"
	"github.com/alienrobotwizard/flotilla-os/core/state"
	"sync"
)

type Worker interface {
	Initialize(conf *config.Config, sm state.Manager, engine engines.Engine) error
	Run(ctx context.Context, wg *sync.WaitGroup) error
}
