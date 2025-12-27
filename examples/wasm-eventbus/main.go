package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
	// Create app
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatal(err)
	}

	// Deploy EventBus WebSocket bridge verticle
	app.DeployVerticle(NewEventBusWSVerticle())

	// Deploy HTTP server for serving static files and WebSocket
	app.DeployVerticle(NewHTTPServerVerticle())

	log.Println("🚀 WASM EventBus Example Server")
	log.Println("   Server: http://localhost:8080")
	log.Println("   WebSocket: ws://localhost:8080/eventbus/ws")

	app.Start()
}

// EventBusWSVerticle provides WebSocket bridge for EventBus
type EventBusWSVerticle struct {
	*core.BaseVerticle
	bridge *core.WebSocketEventBusBridge
}

func NewEventBusWSVerticle() *EventBusWSVerticle {
	return &EventBusWSVerticle{
		BaseVerticle: core.NewBaseVerticle("eventbus-ws"),
	}
}

func (v *EventBusWSVerticle) Start(ctx core.FluxorContext) error {
	// Create WebSocket bridge
	v.bridge = core.NewWebSocketEventBusBridge(ctx.EventBus())

	// Register some test consumers
	eventBus := ctx.EventBus()

	// Test consumer for user.created
	eventBus.Consumer("user.created").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var data map[string]interface{}
		if err := msg.DecodeBody(&data); err != nil {
			return err
		}
		log.Printf("📨 Received user.created: %v", data)
		return nil
	})

	// Test consumer for echo
	eventBus.Consumer("echo").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var data interface{}
		if err := msg.DecodeBody(&data); err != nil {
			return err
		}
		log.Printf("📨 Received echo: %v", data)
		// Echo back
		return msg.Reply(data)
	})

	return nil
}

// HTTPServerVerticle serves static files and WebSocket
type HTTPServerVerticle struct {
	*core.BaseVerticle
	server    *http.Server
	bridge    *core.WebSocketEventBusBridge
	staticDir string
}

func NewHTTPServerVerticle() *HTTPServerVerticle {
	return &HTTPServerVerticle{
		BaseVerticle: core.NewBaseVerticle("http-server"),
		staticDir:    "examples/wasm-eventbus",
	}
}

func (v *HTTPServerVerticle) Start(ctx core.FluxorContext) error {
	// Create WebSocket bridge
	v.bridge = core.NewWebSocketEventBusBridge(ctx.EventBus())

	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/eventbus/ws", v.bridge.HandleWebSocket)

	// Serve static files
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		filePath := filepath.Join(v.staticDir, path)
		http.ServeFile(w, r, filePath)
	})

	v.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := v.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

func (v *HTTPServerVerticle) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Close()
	}
	return nil
}
