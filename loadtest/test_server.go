//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func main() {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	config := web.DefaultFastHTTPServerConfig(":8080")
	server := web.NewFastHTTPServer(gocmd, config)

	router := server.FastRouter()
	router.GETFast("/health", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"ready": true})
	})

	router.GETFast("/api/status", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok", "time": time.Now().Unix()})
	})

	router.POSTFast("/api/echo", func(ctx *web.FastRequestContext) error {
		var data map[string]interface{}
		if err := ctx.BindJSON(&data); err != nil {
			return ctx.JSON(400, map[string]interface{}{"error": "invalid json"})
		}
		return ctx.JSON(200, map[string]interface{}{"echo": data})
	})

	server.SetHandler(func(ctx *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			RequestCtx: ctx,
			GoCMD:      gocmd,
			EventBus:   gocmd.EventBus(),
			Params:     make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	})

	fmt.Println("Starting server on :8080")
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
