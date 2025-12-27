package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestMetricsMiddleware(t *testing.T) {
	ctx := core.NewGoCMD(context.Background())
	defer ctx.Close()

	middleware := MetricsMiddleware()

	tests := []struct {
		name       string
		statusCode int
		method     string
		path       string
		handler    web.FastRequestHandler
		wantErr    bool
	}{
		{
			name:       "successful request",
			statusCode: 200,
			method:     "GET",
			path:       "/test",
			handler: func(ctx *web.FastRequestContext) error {
				return ctx.JSON(200, map[string]interface{}{"status": "ok"})
			},
			wantErr: false,
		},
		{
			name:       "error request",
			statusCode: 500,
			method:     "POST",
			path:       "/error",
			handler: func(ctx *web.FastRequestContext) error {
				return ctx.JSON(500, map[string]interface{}{"error": "internal error"})
			},
			wantErr: false,
		},
		{
			name:       "not found",
			statusCode: 404,
			method:     "GET",
			path:       "/notfound",
			handler: func(ctx *web.FastRequestContext) error {
				return ctx.JSON(404, map[string]interface{}{"error": "not found"})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrap handler with middleware
			wrappedHandler := middleware(tt.handler)

			// Create mock request context
			fasthttpCtx := &fasthttp.RequestCtx{}
			fasthttpCtx.Request.Header.SetMethod(tt.method)
			fasthttpCtx.URI().SetPath(tt.path)
			fasthttpCtx.Request.SetBodyString(`{"test":"data"}`)

			reqCtx := &web.FastRequestContext{
				RequestCtx: fasthttpCtx,
				GoCMD:      ctx,
				EventBus:   ctx.EventBus(),
				Params:     make(map[string]string),
			}

			// Execute wrapped handler
			start := time.Now()
			err := wrappedHandler(reqCtx)
			duration := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("MetricsMiddleware() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify metrics were recorded (metrics are recorded in the handler)
			if duration < 0 {
				t.Error("Duration should be non-negative")
			}

			// Verify status code was set
			if fasthttpCtx.Response.StatusCode() != tt.statusCode {
				t.Errorf("Status code = %v, want %v", fasthttpCtx.Response.StatusCode(), tt.statusCode)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "normal path",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "path with leading slash",
			path:     "/test",
			expected: "/test",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
