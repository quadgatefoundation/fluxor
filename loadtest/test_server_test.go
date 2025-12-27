package main

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestTestServerSetup(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := web.DefaultFastHTTPServerConfig(":0") // Use :0 for random port
	server := web.NewFastHTTPServer(gocmd, config)

	if server == nil {
		t.Fatal("NewFastHTTPServer() returned nil")
	}

	router := server.FastRouter()
	if router == nil {
		t.Fatal("FastRouter() returned nil")
	}

	// Test route registration
	router.GETFast("/health", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"ready": true})
	})

	router.GETFast("/api/status", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.POSTFast("/api/echo", func(ctx *web.FastRequestContext) error {
		var data map[string]interface{}
		if err := ctx.BindJSON(&data); err != nil {
			return ctx.JSON(400, map[string]interface{}{"error": "invalid json"})
		}
		return ctx.JSON(200, map[string]interface{}{"echo": data})
	})

	// Verify server can be configured
	server.SetHandler(func(ctx *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			RequestCtx: ctx,
			GoCMD:      gocmd,
			EventBus:   gocmd.EventBus(),
			Params:     make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	})

	// Test cleanup
	if err := server.Stop(); err != nil {
		t.Errorf("server.Stop() error = %v", err)
	}
}
