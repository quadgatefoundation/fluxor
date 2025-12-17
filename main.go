package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxor-io/fluxor/pkg/component"
	"github.com/fluxor-io/fluxor/pkg/runtime"
	"github.com/fluxor-io/fluxor/pkg/types"
)

func main() {
	// Create a new runtime.
	opts := runtime.Options{
		MailboxSize: 1024,
		NumWorkers:  8,
		QueueSize:   1024,
	}
	rt := runtime.NewRuntime(opts)

	// Start the runtime.
	if err := rt.Start(context.Background()); err != nil {
		panic(err)
	}

	// Create and deploy the Gemini component.
	gemini := &component.Gemini{}
	if err := rt.Deploy(context.Background(), gemini); err != nil {
		panic(err)
	}

	// Send a prompt to the Gemini component and wait for the response.
	go func() {
		time.Sleep(3 * time.Second) // Wait for the component to start
		prompt := "Tell me a story about a brave robot."
		log.Printf("Sending prompt: %s", prompt)
		resp, err := rt.Bus().Request(context.Background(), "/gemini/generate", types.Message{Payload: prompt})
		if err != nil {
			log.Printf("Error sending prompt: %v", err)
			return
		}
		log.Printf("Response: %s", resp.Payload)
	}()

	// Wait for a signal to shut down.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Shut down the runtime gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rt.Stop(ctx); err != nil {
		panic(err)
	}
}
