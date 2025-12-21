package core

import (
	"context"
	"log/slog"
)

type FluxorContext struct {
	id     string
	bus    *Bus
	worker *WorkerPool
	stdCtx context.Context
}

func NewFluxorContext(stdCtx context.Context, bus *Bus, wp *WorkerPool, id string) *FluxorContext {
	return &FluxorContext{id: id, bus: bus, worker: wp, stdCtx: stdCtx}
}

func (c *FluxorContext) ID() string             { return c.id }
func (c *FluxorContext) Bus() *Bus              { return c.bus }
func (c *FluxorContext) Worker() *WorkerPool    { return c.worker }
func (c *FluxorContext) Log() *slog.Logger      { return slog.Default().With("id", c.id) }
func (c *FluxorContext) Ctx() context.Context   { return c.stdCtx }
