package fx

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/valyala/fasthttp"
)

// FastContext is a fasthttp-friendly context wrapper for lite-fast.
// It is intentionally minimal and allocation-conscious.
type FastContext struct {
	RC      *fasthttp.RequestCtx
	Params  map[string]string
	coreCtx *core.FluxorContext
}

func NewFastContext(rc *fasthttp.RequestCtx, params map[string]string, cCtx *core.FluxorContext) *FastContext {
	return &FastContext{
		RC:      rc,
		Params:  params,
		coreCtx: cCtx,
	}
}

func (c *FastContext) JSON(code int, data any) error {
	c.RC.SetStatusCode(code)
	c.RC.SetContentType("application/json")

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, werr := c.RC.Write(b); werr != nil {
		return werr
	}
	return nil
}

func (c *FastContext) Ok(data any) error {
	return c.JSON(fasthttp.StatusOK, data)
}

func (c *FastContext) Error(code int, msg string) error {
	// Safe default error body (don’t reflect internal errors)
	return c.JSON(code, JSON{"error": msg})
}

func (c *FastContext) Text(code int, text string) error {
	c.RC.SetStatusCode(code)
	c.RC.SetContentType("text/plain; charset=utf-8")
	if _, err := c.RC.WriteString(text); err != nil {
		return err
	}
	return nil
}

func (c *FastContext) Query(key string) string {
	return string(c.RC.QueryArgs().Peek(key))
}

func (c *FastContext) Header(key string) string {
	return string(c.RC.Request.Header.Peek(key))
}

func (c *FastContext) SetHeader(key, value string) {
	c.RC.Response.Header.Set(key, value)
}

func (c *FastContext) Param(key string) string {
	if c.Params == nil {
		return ""
	}
	return c.Params[key]
}

func (c *FastContext) Log() *slog.Logger {
	l := c.coreCtx.Log()
	return l.With(
		"method", string(c.RC.Method()),
		"path", string(c.RC.Path()),
	)
}

func (c *FastContext) Worker() *core.WorkerPool  { return c.coreCtx.Worker() }
func (c *FastContext) Bus() *core.Bus            { return c.coreCtx.Bus() }
func (c *FastContext) Core() *core.FluxorContext { return c.coreCtx }

// Fail-fast helpers
func (c *FastContext) MustParam(key string) (string, error) {
	v := c.Param(key)
	if v == "" {
		return "", fmt.Errorf("missing param: %s", key)
	}
	return v, nil
}
