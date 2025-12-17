package core

import (
	"sync"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan any 
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan any),
	}
}

// Subscribe returns a typed channel.
func Subscribe[T any](eb *EventBus, topic string) <-chan T {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan any, 100) 
	eb.subscribers[topic] = append(eb.subscribers[topic], ch)

	out := make(chan T, 100)
	go func() {
		defer close(out)
		for msg := range ch {
			if typedMsg, ok := msg.(T); ok {
				out <- typedMsg
			}
		}
	}()
	return out
}

// Publish sends a message non-blocking.
func (eb *EventBus) Publish(topic string, payload any) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if subs, ok := eb.subscribers[topic]; ok {
		for _, ch := range subs {
			select {
			case ch <- payload:
			default:
				// Dropping message if consumer is slow (Backpressure strategy needed later)
			}
		}
	}
}
