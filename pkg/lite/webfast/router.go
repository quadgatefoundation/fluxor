package webfast

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/valyala/fasthttp"
)

type HandlerFunc func(c *fx.FastContext) error
type Middleware func(next HandlerFunc) HandlerFunc

type route struct {
	method     []byte
	pattern    string
	segments   []segment
	handler    HandlerFunc
	middleware []Middleware
	hasParams  bool
}

type segment struct {
	static []byte
	param  string // if non-empty, this segment captures into param
}

type Router struct {
	routes     []*route
	middleware []Middleware
	notFound   HandlerFunc
	onError    func(c *fx.FastContext, err error) error

	coreCtx *core.FluxorContext

	paramPool sync.Pool
}

func NewRouter() *Router {
	r := &Router{
		routes:     make([]*route, 0, 32),
		middleware: make([]Middleware, 0),
	}
	r.notFound = func(c *fx.FastContext) error {
		return c.Error(fasthttp.StatusNotFound, "Not Found")
	}
	r.onError = func(c *fx.FastContext, err error) error {
		c.Log().Error("handler error", "err", err)
		return c.Error(fasthttp.StatusInternalServerError, "Internal Server Error")
	}
	return r
}

// Bind attaches the runtime context (required before serving requests).
func (r *Router) Bind(coreCtx *core.FluxorContext) {
	r.coreCtx = coreCtx
}

func (r *Router) Use(mw ...Middleware) { r.middleware = append(r.middleware, mw...) }

func (r *Router) GET(path string, h HandlerFunc, mw ...Middleware) {
	r.add([]byte(http.MethodGet), path, h, mw...)
}
func (r *Router) POST(path string, h HandlerFunc, mw ...Middleware) {
	r.add([]byte(http.MethodPost), path, h, mw...)
}

func (r *Router) add(method []byte, pattern string, h HandlerFunc, mw ...Middleware) {
	segs, hasParams := compilePattern(pattern)
	r.routes = append(r.routes, &route{
		method:     method,
		pattern:    pattern,
		segments:   segs,
		handler:    h,
		middleware: append([]Middleware(nil), mw...),
		hasParams:  hasParams,
	})
}

func (r *Router) Handler() fasthttp.RequestHandler {
	return func(rc *fasthttp.RequestCtx) {
		// Params map is allocated only when we match a param route.
		c := fx.NewFastContext(rc, nil, r.coreCtx)

		method := rc.Method()
		path := rc.Path()

		for _, rt := range r.routes {
			if !bytes.Equal(rt.method, method) {
				continue
			}

			if ok := matchAndFill(rt, path, c, &r.paramPool); !ok {
				continue
			}

			h := rt.handler
			for i := len(rt.middleware) - 1; i >= 0; i-- {
				h = rt.middleware[i](h)
			}
			for i := len(r.middleware) - 1; i >= 0; i-- {
				h = r.middleware[i](h)
			}

			if err := h(c); err != nil {
				_ = r.onError(c, err)
			}

			// Return params map to pool (if used)
			if c.Params != nil {
				clear(c.Params)
				r.paramPool.Put(c.Params)
			}
			return
		}

		_ = r.notFound(c)
	}
}

func compilePattern(pattern string) ([]segment, bool) {
	// Split on '/', ignore leading empty segment.
	var out []segment
	hasParams := false

	p := []byte(pattern)
	i := 0
	for i < len(p) && p[i] == '/' {
		i++
	}
	start := i
	for i <= len(p) {
		if i == len(p) || p[i] == '/' {
			if i > start {
				part := p[start:i]
				if len(part) > 0 && part[0] == ':' && len(part) > 1 {
					hasParams = true
					out = append(out, segment{param: string(part[1:])})
				} else {
					out = append(out, segment{static: append([]byte(nil), part...)})
				}
			} else if i == start {
				// double slash or trailing slash -> treat as empty segment (unsupported)
				out = append(out, segment{static: []byte{}})
			}
			i++
			start = i
			continue
		}
		i++
	}
	return out, hasParams
}

func matchAndFill(rt *route, path []byte, c *fx.FastContext, pool *sync.Pool) bool {
	// Fast exact match for patterns without params.
	if !rt.hasParams && bytes.Equal(path, []byte(rt.pattern)) {
		return true
	}

	// Split path into segments without allocations.
	// Reject paths with empty segments for simplicity.
	if rt.hasParams {
		if m, ok := pool.Get().(map[string]string); ok {
			c.Params = m
		} else {
			c.Params = make(map[string]string, 4)
		}
	} else {
		c.Params = nil
	}

	i := 0
	for i < len(path) && path[i] == '/' {
		i++
	}
	segIdx := 0
	start := i
	for i <= len(path) {
		if segIdx >= len(rt.segments) {
			return false
		}
		if i == len(path) || path[i] == '/' {
			if i == start {
				return false
			}
			part := path[start:i]
			seg := rt.segments[segIdx]
			if seg.param != "" {
				// NOTE: this allocates; only happens for param routes.
				c.Params[seg.param] = string(part)
			} else {
				if !bytes.Equal(seg.static, part) {
					return false
				}
			}
			segIdx++
			i++
			start = i
			continue
		}
		i++
	}

	return segIdx == len(rt.segments)
}
