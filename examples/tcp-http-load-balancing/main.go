package main

import (
	"flag"
	"log"

	"github.com/fluxorio/fluxor/examples/tcp-http-load-balancing/verticles"
	"github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
	// Parse command line flags
	httpAddr := flag.String("http", ":8080", "HTTP server address")
	tcpAddr := flag.String("tcp", ":9090", "TCP server address")
	workerCount := flag.Int("workers", 2, "Number of worker nodes (minimum 2)")
	flag.Parse()

	// Ensure at least 2 workers
	if *workerCount < 2 {
		*workerCount = 2
	}

	log.Printf("Starting Load Balancing TCP/HTTP Server")
	log.Printf("Configuration:")
	log.Printf("  HTTP Address: %s", *httpAddr)
	log.Printf("  TCP Address:  %s", *tcpAddr)
	log.Printf("  Workers:      %d", *workerCount)

	// Create MainVerticle (primary pattern)
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatalf("Failed to create MainVerticle: %v", err)
	}

	// Generate worker IDs
	workerIDs := make([]string, *workerCount)
	for i := 0; i < *workerCount; i++ {
		workerIDs[i] = string(rune('A' + i)) // A, B, C, ...
	}

	// Deploy Workers first (dependencies first pattern)
	log.Println("Deploying workers...")
	for _, id := range workerIDs {
		worker := verticles.NewWorkerVerticle(id)
		if _, err := app.DeployVerticle(worker); err != nil {
			log.Fatalf("Failed to deploy worker %s: %v", id, err)
		}
		log.Printf("  Worker %s deployed", id)
	}

	// Deploy Master (load balancer)
	log.Println("Deploying master...")
	master := verticles.NewMasterVerticle(workerIDs, *httpAddr, *tcpAddr)
	if _, err := app.DeployVerticle(master); err != nil {
		log.Fatalf("Failed to deploy master: %v", err)
	}
	log.Println("  Master deployed")

	// Print usage information
	log.Println("")
	log.Println("=== Load Balancing Server Started ===")
	log.Println("")
	log.Println("HTTP Endpoints:")
	log.Printf("  GET  http://localhost%s/health     - Health check", *httpAddr)
	log.Printf("  GET  http://localhost%s/status     - Detailed status", *httpAddr)
	log.Printf("  GET  http://localhost%s/workers    - List workers", *httpAddr)
	log.Printf("  GET  http://localhost%s/process?data=hello - Process request", *httpAddr)
	log.Printf("  POST http://localhost%s/process    - Process JSON request", *httpAddr)
	log.Println("")
	log.Println("TCP Protocol (line-based):")
	log.Printf("  echo 'hello' | nc localhost %s", (*tcpAddr)[1:])
	log.Printf("  echo 'PING' | nc localhost %s", (*tcpAddr)[1:])
	log.Printf("  echo 'STATUS' | nc localhost %s", (*tcpAddr)[1:])
	log.Println("")
	log.Println("Architecture:")
	log.Println("  1 Master Node (Load Balancer)")
	log.Printf("  %d Worker Nodes (A, B, ...)", *workerCount)
	log.Println("")
	log.Println("Press Ctrl+C to stop")

	// Start application (blocks until SIGINT/SIGTERM)
	if err := app.Start(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
