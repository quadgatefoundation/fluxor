package core

import (
	"context"
	"log/slog"
)

// RuntimeRef is a read-only interface to the Runtime,
// preventing reactors from calling dangerous methods like Shutdown().
type RuntimeRef interface {
	EventBus() *EventBus
}

type FluxorContext struct {
	id      string         // Deployment ID (UUID)
	config  map[string]any // Configuration map
	runtime RuntimeRef     // Reference to parent runtime
	stdCtx  context.Context
}

func NewFluxorContext(stdCtx context.Context, rt RuntimeRef, id string, conf map[string]any) *FluxorContext {
	return &FluxorContext{
		id:      id,
		runtime: rt,
		stdCtx:  stdCtx,
		config:  conf,
	}
}

// Accessors
func (c *FluxorContext) ID() string             { return c.id }
func (c *FluxorContext) Ctx() context.Context   { return c.stdCtx }
func (c *FluxorContext) Config() map[string]any { return c.config }

// Bus returns the event bus from the runtime.
func (c *FluxorContext) Bus() *EventBus {
	return c.runtime.EventBus()
}

// Log returns a structured logger with the deployment ID already added.
func (c *FluxorContext) Log() *slog.Logger {
	return slog.With("deploymentID", c.id)
}
