package workers

import (
	"context"
	"sync"
)

type Worker interface {
	Run(ctx context.Context, wg *sync.WaitGroup) error
}
