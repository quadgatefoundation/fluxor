package reactor

import (
	"context"
	"errors"
	"time"
)

var ErrBackpressure = errors.New("backpressure: queue full")

type Reactor struct {
	mailbox chan func()
}

type ReactorOptions struct {
	MailboxSize int
}

func NewReactor(opts ReactorOptions) *Reactor {
	if opts.MailboxSize <= 0 {
		opts.MailboxSize = 1024 // Default mailbox size
	}
	return &Reactor{
		mailbox: make(chan func(), opts.MailboxSize),
	}
}

func (r *Reactor) Start() {
	go r.run()
}

func (r *Reactor) run() {
	for fn := range r.mailbox {
		fn()
	}
}

func (r *Reactor) Stop(ctx context.Context) error {
	close(r.mailbox)
	return nil
}

func (r *Reactor) Post(fn func()) error {
	select {
	case r.mailbox <- fn:
		return nil
	default:
		return ErrBackpressure
	}
}

func (r *Reactor) PostTimeout(d time.Duration, fn func()) error {
	timer := time.NewTimer(d)
	select {
	case r.mailbox <- fn:
		timer.Stop()
		return nil
	case <-timer.C:
		return ErrBackpressure
	}
}

func (r *Reactor) SetTimer(d time.Duration, fn func()) func() {
	timer := time.NewTimer(d)
	go func() {
		<-timer.C
		r.Post(fn)
	}()
	return func() { timer.Stop() }
}

func (r *Reactor) SetPeriodic(d time.Duration, fn func()) func() {
	ticker := time.NewTicker(d)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				r.Post(fn)
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}
