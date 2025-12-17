package component

import (
	"context"
	"sync"
	"github.com/fluxor-io/fluxor/pkg/types"
)

// Base is a helper struct that provides a robust implementation of the Component interface.
// It is intended to be embedded in other components to provide common lifecycle management
// and goroutine supervision.
type Base struct {
	wg         sync.WaitGroup
	stopCtx    context.Context
	stopCancel context.CancelFunc
}


// OnStart is a lifecycle hook that is called when the component is started.
// Components that embed Base can implement this method to perform their own startup logic.
func (b *Base) OnStart(ctx context.Context, bus types.Bus) error {
	b.stopCtx, b.stopCancel = context.WithCancel(context.Background())
	// By default, do nothing.
	return nil
}

// OnStop is a lifecycle hook that is called when the component is stopped.
// Components that embed Base can implement this method to perform their own shutdown logic.
func (b *Base) OnStop(ctx context.Context) error {
	if b.stopCancel != nil {
		b.stopCancel()
	}
	b.wg.Wait()
	return nil
}

// Go starts a new goroutine that is supervised by the component.
// The component's Stop method will wait for all supervised goroutines to complete.
func (b *Base) Go(f func(ctx context.Context)) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		if b.stopCtx != nil {
			f(b.stopCtx)
		}
	}()
}
