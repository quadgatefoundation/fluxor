package reactor

import (
	"context"

	"github.com/fluxor-io/fluxor/pkg/types"
)

type Reactor struct {
	name    string
	mailbox chan func()
	bus     types.Bus
}

func NewReactor(name string, size int) *Reactor {
	r := &Reactor{
		name:    name,
		mailbox: make(chan func(), size),
	}
	return r
}

func (r *Reactor) Name() string {
	return r.name
}

func (r *Reactor) OnStart(ctx context.Context, bus types.Bus) error {
	r.bus = bus
	go r.loop(ctx)
	return nil
}

func (r *Reactor) OnStop(ctx context.Context) error {
	// A more robust implementation would handle draining the mailbox
	// and ensuring the loop exits gracefully.
	return nil
}

// Execute submits a function for execution on the reactor's event loop.
// It returns types.ErrBackpressure if the reactor's mailbox is full.
func (r *Reactor) Execute(fn func()) error {
	select {
	case r.mailbox <- fn:
		return nil
	default:
		return types.ErrBackpressure
	}
}

func (r *Reactor) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case fn := <-r.mailbox:
			r.safeExecute(fn)
		}
	}
}

func (r *Reactor) safeExecute(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			// TODO: Log the panic. A component panic should not bring down the reactor.
		}
	}()
	fn()
}
