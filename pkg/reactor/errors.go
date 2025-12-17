package reactor

import "errors"

var (
	// ErrBackpressure is returned when a reactor's mailbox is full.
	ErrBackpressure = errors.New("reactor: backpressure")

	// ErrStopped is returned when a post is attempted on a stopped reactor.
	ErrStopped = errors.New("reactor: stopped")
)
