package reactor

import (
	"context"
	"sync"
)

// Reactor is a single-goroutine event loop with a bounded mailbox.
type Reactor struct {
	mailbox chan func()
	stopped chan struct{}
	once    sync.Once
}

// New creates a new Reactor with the given mailbox size.
func New(mailboxSize int) *Reactor {
	return &Reactor{
		mailbox: make(chan func(), mailboxSize),
		stopped: make(chan struct{}),
	}
}

// Start begins the reactor's event loop.
func (r *Reactor) Start() {
	r.once.Do(func() {
		go r.run()
	})
}

// Stop gracefully shuts down the reactor.
func (r *Reactor) Stop(ctx context.Context) error {
	close(r.mailbox)
	select {
	case <-r.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Post adds a function to the reactor's mailbox for execution.
// It is non-blocking and returns ErrBackpressure if the mailbox is full.
// It returns ErrStopped if the reactor is not running or has been stopped.
func (r *Reactor) Post(fn func()) error {
	select {
	case <-r.stopped:
		return ErrStopped
	default:
	}

	select {
	case r.mailbox <- fn:
		return nil
	default:
		return ErrBackpressure
	}
}

func (r *Reactor) run() {
	defer close(r.stopped)
	for fn := range r.mailbox {
		fn()
	}
}
