package main

import (
	"context"
	"fmt"
	"github.com/example/goreactor/pkg/gateway"
	"github.com/example/goreactor/pkg/runtime"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	rt := runtime.NewRuntime(4) // 4 reactors
	rt.Start()

	// Deploy gateway component
	gw := gateway.NewGateway()
	_, err := rt.Deploy(ctx, gw)
	if err != nil {
		log.Fatalf("Failed to deploy gateway: %v", err)
	}

	fmt.Println("Gateway started on :8080")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down gateway...")
	cancel()
	rt.Stop(ctx)
	log.Println("Gateway shut down gracefully")
}
