package main

import (
	"github.com/khangdcicloud/fluxor/pkg/fluxor"
	"github.com/khangdcicloud/fluxor/pkg/fx"
	"github.com/khangdcicloud/fluxor/pkg/web"
)

func main() {
	// 1. Runtime
	app := fluxor.New()

	// 2. Logic
	r := web.NewRouter()
	
r.GET("/api/ping", func(c *fx.Context) error {
		return c.Ok(fx.JSON{"msg": "pong", "framework": "Fluxor"})
	})

	r.GET("/api/work", func(c *fx.Context) error {
		// Test Worker Pool & Bus
		c.Worker().Submit(func() {
			// Giả lập việc nặng
			result := "Heavy Task Done"
			c.Bus().Publish("log", result)
		})
		return c.Ok(fx.JSON{"status": "processing"})
	})

	// 3. Component
	v := web.NewHttpVerticle("8080", r)

	// 4. Deploy & Run
	app.Deploy(v)
	app.Run()
}
