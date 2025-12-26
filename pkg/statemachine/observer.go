package statemachine

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
)

// LoggingObserver logs all state transitions.
type LoggingObserver struct {
	logger interface {
		Info(...interface{})
		Error(...interface{})
	}
}

// NewLoggingObserver creates a new logging observer.
func NewLoggingObserver(logger interface {
	Info(...interface{})
	Error(...interface{})
}) *LoggingObserver {
	return &LoggingObserver{logger: logger}
}

func (o *LoggingObserver) OnTransition(ctx context.Context, from string, to string, event Event) {
	o.logger.Info(fmt.Sprintf("State transition: %s -> %s (event: %s)", from, to, event.Name))
}

func (o *LoggingObserver) OnError(ctx context.Context, err error) {
	o.logger.Error(fmt.Sprintf("State machine error: %v", err))
}

// MetricsObserver tracks state machine metrics.
type MetricsObserver struct {
	transitionCount map[string]int // from:to -> count
	eventCount      map[string]int // event -> count
	errorCount      int
}

// NewMetricsObserver creates a new metrics observer.
func NewMetricsObserver() *MetricsObserver {
	return &MetricsObserver{
		transitionCount: make(map[string]int),
		eventCount:      make(map[string]int),
	}
}

func (o *MetricsObserver) OnTransition(ctx context.Context, from string, to string, event Event) {
	key := fmt.Sprintf("%s:%s", from, to)
	o.transitionCount[key]++
	o.eventCount[event.Name]++
}

func (o *MetricsObserver) OnError(ctx context.Context, err error) {
	o.errorCount++
}

// GetMetrics returns the collected metrics.
func (o *MetricsObserver) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"transitions": o.transitionCount,
		"events":      o.eventCount,
		"errors":      o.errorCount,
	}
}

// EventBusObserver publishes state changes to EventBus.
type EventBusObserver struct {
	eventBus core.EventBus
	prefix   string
}

// NewEventBusObserver creates a new EventBus observer.
func NewEventBusObserver(eventBus core.EventBus, prefix string) *EventBusObserver {
	return &EventBusObserver{
		eventBus: eventBus,
		prefix:   prefix,
	}
}

func (o *EventBusObserver) OnTransition(ctx context.Context, from string, to string, event Event) {
	address := fmt.Sprintf("%s.transition", o.prefix)
	o.eventBus.Publish(address, StateChangeEvent{
		From:      from,
		To:        to,
		Event:     event.Name,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

func (o *EventBusObserver) OnError(ctx context.Context, err error) {
	address := fmt.Sprintf("%s.error", o.prefix)
	o.eventBus.Publish(address, map[string]interface{}{
		"error": err.Error(),
	})
}

// ChainObserver chains multiple observers.
type ChainObserver struct {
	observers []Observer
}

// NewChainObserver creates a new chain observer.
func NewChainObserver(observers ...Observer) *ChainObserver {
	return &ChainObserver{observers: observers}
}

func (o *ChainObserver) OnTransition(ctx context.Context, from string, to string, event Event) {
	for _, observer := range o.observers {
		observer.OnTransition(ctx, from, to, event)
	}
}

func (o *ChainObserver) OnError(ctx context.Context, err error) {
	for _, observer := range o.observers {
		observer.OnError(ctx, err)
	}
}
