package gateway

import (
	"context"
	"fmt"
	"github.com/example/goreactor/pkg/httpx"
	"log"
	"net/http"
	"time"
)

type Gateway struct {
	server *http.Server
}

func NewGateway() *Gateway {
	g := &Gateway{}

	// Use httpx.NewHandler from our shared package
	handler := httpx.NewHandler(g.handleRequest)

	g.server = &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	return g
}

func (g *Gateway) Start(ctx context.Context) error {
	log.Println("Starting gateway server on :8080")
	go func() {
		if err := g.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start gateway server: %v", err)
		}
	}()
	return nil
}

func (g *Gateway) Stop(ctx context.Context) error {
	log.Println("Stopping gateway server")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return g.server.Shutdown(ctx)
}

func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	// Simple echo response for now
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Hello from the gateway!")
}
