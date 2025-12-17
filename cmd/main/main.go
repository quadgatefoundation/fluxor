package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/component"
	"github.com/fluxor-io/fluxor/pkg/httpx"
	"github.com/fluxor-io/fluxor/pkg/runtime"
)

// GreeterService is a component that provides a greeting service.
//
// It demonstrates the core principles of a Fluxor component:
//   - It embeds component.Base to inherit lifecycle management.
//   - It interacts with other components via the event bus.
//   - Its logic is executed on a dedicated reactor, ensuring sequential, non-blocking execution.
//
// This service listens for messages on the "/greet" topic, extracts a name from the
// request payload, and sends a personalized greeting back to the originator.
type GreeterService struct {
	component.Base
	bus bus.Bus
}

// OnStart is called by the runtime when the component is started.
// It registers the event bus consumer for the "/greet" topic.
func (g *GreeterService) OnStart(ctx context.Context, b bus.Bus) {
	g.bus = b
	// Registering the handler tells the bus to execute g.handleGreet
	// whenever a message is received on the "/greet" topic.
	g.bus.Consumer("/greet", g.handleGreet)
	log.Println("GreeterService started and listening on topic '/greet'")
}

// handleGreet is the message handler for the "/greet" topic.
// It is always executed on the component's assigned reactor goroutine.
func (g *GreeterService) handleGreet(msg bus.Message) {
	// The payload from the httpx gateway is an HttpRequest struct.
	req := msg.Payload.(*httpx.HttpRequest)
	u, _ := url.Parse(req.URL)
	name := u.Query().Get("name")
	if name == "" {
		name = "world"
	}

	// The message includes a ReplyTo topic for the response.
	// Sending the reply back to this topic completes the request-reply cycle.
	if msg.ReplyTo != "" {
		g.bus.Send(msg.ReplyTo, bus.Message{
			Payload:       fmt.Sprintf("hello %s", name),
			CorrelationID: msg.CorrelationID,
		})
	}
}

func main() {
	// Create a new in-process event bus.
	b := bus.NewBus()

	// Create a new runtime.
	rt := runtime.NewRuntime(b)

	// Register components with the runtime.
	// The "gateway" is the HTTP server that acts as an entry point.
	// The "greeter" is our business logic component.
	rt.Register("gateway", httpx.NewServer(8080, b))
	rt.Register("greeter", &GreeterService{})

	// Start the runtime in a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("failed to start runtime: %v", err)
	}

	log.Println("Runtime started. Gateway listening on :8080")
	log.Println("Try visiting http://localhost:8080/greet?name=Fluxor")

	// Set up a channel to listen for OS signals for graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")

	// Stop the runtime gracefully with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := rt.Stop(shutdownCtx); err != nil {
		log.Fatalf("failed to stop runtime: %v", err)
	}

	log.Println("Runtime stopped")
}
