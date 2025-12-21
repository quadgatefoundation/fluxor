package web

import "github.com/khangdcicloud/fluxor/pkg/fx"

type HandlerFunc func(c *fx.Context) error

type Router struct {
	routes map[string]HandlerFunc
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]HandlerFunc)}
}

func (r *Router) GET(path string, h HandlerFunc) {
	r.routes[path] = h
}

// Internal use
func (r *Router) Handle(c *fx.Context) error {
	if h, ok := r.routes[c.R.URL.Path]; ok {
		return h(c)
	}
	return c.Error(404, "Not Found")
}
