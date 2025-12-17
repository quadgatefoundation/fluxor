package types

import (
	"context"
	"errors"
)

var ErrBackpressure = errors.New("backpressure")

// Message represents a message passed on the event bus.
type Message struct {
	Topic         string
	Payload       interface{}
	ReplyTo       string
	CorrelationID string
}

// Mailbox is a channel for receiving messages.
type Mailbox chan Message

// Bus is the interface for the event bus.
type Bus interface {
	// Publish sends a message to all subscribers of a topic.
	Publish(topic string, msg Message)

	// Subscribe adds a handler for a topic.
	Subscribe(topic, componentName string, handler Mailbox) error

	// Unsubscribe removes a handler for a topic.
	Unsubscribe(topic, componentName string, handler Mailbox) error

	// Send sends a message to one subscriber of a topic.
	Send(topic string, msg Message) error

	// Request sends a message to one subscriber of a topic and waits for a reply.
	Request(ctx context.Context, topic string, msg Message) (Message, error)
}

// Component is the interface for a deployable unit.
type Component interface {
	// Name returns the name of the component.
	Name() string

	// OnStart is called when the component is started.
	OnStart(ctx context.Context, bus Bus) error

	// OnStop is called when the component is stopped.
	OnStop(ctx context.Context) error
}
