package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/statemachine"
)

func main() {
	// Create Fluxor app
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatal(err)
	}

	// Create state machine verticle with HTTP API
	smVerticle := statemachine.NewStateMachineVerticle(&statemachine.StateMachineVerticleConfig{
		HTTPAddr: ":8082",
	})

	// Deploy the state machine verticle
	app.DeployVerticle(smVerticle)

	// Register an order processing state machine definition
	orderDef := buildOrderStateMachine()
	if err := smVerticle.RegisterDefinition(orderDef); err != nil {
		log.Fatal(err)
	}

	log.Println("State machine verticle deployed with HTTP API on :8082")
	log.Println("Example: Create a machine and send events via HTTP")
	log.Println("  POST http://localhost:8082/machines {\"definitionId\": \"order-processing\"}")
	log.Println("  POST http://localhost:8082/machines/{id}/events {\"name\": \"approve\", \"data\": {\"orderId\": \"123\"}}")

	// Create a sample machine and demonstrate usage
	go demonstrateStateMachine(smVerticle)

	// Start the app (blocks until SIGINT/SIGTERM)
	app.Start()
}

func buildOrderStateMachine() *statemachine.Definition {
	def, err := statemachine.NewBuilder("order-processing").
		Name("Order Processing").
		Description("Manages order lifecycle from pending to completion").
		InitialState("pending").
		
		// Pending state
		State("pending").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("📋 Order is pending review...")
				return nil
			}).
			Exit(func(ctx context.Context, event statemachine.Event) error {
				log.Println("✓ Order review completed")
				return nil
			}).
			On("approve", "approved").
				Guard(statemachine.DataFieldExists("orderId")).
				Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
					orderID := event.Data["orderId"]
					log.Printf("✓ Approving order %v...", orderID)
					// Simulate approval logic
					time.Sleep(100 * time.Millisecond)
					return nil
				}).
				Done().
			On("reject", "rejected").
				Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
					reason := event.Data["reason"]
					log.Printf("✗ Rejecting order: %v", reason)
					return nil
				}).
				Done().
			Done().
		
		// Approved state
		State("approved").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("✓ Order approved! Ready for processing.")
				return nil
			}).
			On("process", "processing").
				Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
					log.Println("⚙️  Starting order processing...")
					return nil
				}).
				Done().
			On("cancel", "cancelled").Done().
			Done().
		
		// Processing state
		State("processing").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("⚙️  Processing order...")
				return nil
			}).
			On("complete", "completed").
				Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
					log.Println("📦 Preparing shipment...")
					time.Sleep(200 * time.Millisecond)
					return nil
				}).
				Done().
			On("fail", "failed").Done().
			Done().
		
		// Completed state (final)
		State("completed").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("✅ Order completed successfully!")
				return nil
			}).
			Final(true).
			Done().
		
		// Rejected state (final)
		State("rejected").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("❌ Order rejected")
				return nil
			}).
			Final(true).
			Done().
		
		// Cancelled state (final)
		State("cancelled").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("🚫 Order cancelled")
				return nil
			}).
			Final(true).
			Done().
		
		// Failed state (final)
		State("failed").
			Entry(func(ctx context.Context, event statemachine.Event) error {
				log.Println("💥 Order processing failed")
				return nil
			}).
			Final(true).
			Done().
		
		Build()

	if err != nil {
		log.Fatal(err)
	}

	return def
}

func demonstrateStateMachine(verticle *statemachine.StateMachineVerticle) {
	time.Sleep(2 * time.Second) // Wait for verticle to start

	ctx := context.Background()

	log.Println("\n========================================")
	log.Println("Demonstrating State Machine")
	log.Println("========================================\n")

	// Create a machine instance
	machine, err := verticle.CreateMachine(ctx, "order-processing", "order-123", map[string]interface{}{
		"customerID": "customer-456",
	})
	if err != nil {
		log.Printf("Failed to create machine: %v", err)
		return
	}

	log.Printf("Created state machine: %s\n", machine.ID())
	log.Printf("Current state: %s\n\n", machine.CurrentState())

	// Simulate order approval
	time.Sleep(1 * time.Second)
	log.Println("Sending 'approve' event...")
	err = machine.Send(ctx, statemachine.Event{
		Name: "approve",
		Data: map[string]interface{}{
			"orderId":    "order-123",
			"approvedBy": "admin",
		},
		Timestamp: time.Now(),
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	log.Printf("Current state: %s\n\n", machine.CurrentState())

	// Start processing
	time.Sleep(1 * time.Second)
	log.Println("Sending 'process' event...")
	err = machine.Send(ctx, statemachine.Event{
		Name:      "process",
		Timestamp: time.Now(),
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	log.Printf("Current state: %s\n\n", machine.CurrentState())

	// Complete order
	time.Sleep(1 * time.Second)
	log.Println("Sending 'complete' event...")
	err = machine.Send(ctx, statemachine.Event{
		Name:      "complete",
		Timestamp: time.Now(),
	})
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	log.Printf("Current state: %s\n\n", machine.CurrentState())

	// Print history
	log.Println("========================================")
	log.Println("State Transition History:")
	log.Println("========================================")
	for i, entry := range machine.GetHistory() {
		log.Printf("%d. %s -> %s (event: %s, duration: %v)",
			i+1, entry.From, entry.To, entry.Event, entry.Duration)
	}
	log.Println("\n========================================\n")

	// Demonstrate EventBus integration
	demonstrateEventBusIntegration(machine, ctx)
}

func demonstrateEventBusIntegration(machine statemachine.StateMachine, ctx context.Context) {
	log.Println("Demonstrating EventBus integration...")
	log.Println("State changes are published to: statemachine.{machineId}.transition")
	log.Println("Events can be sent to: statemachine.{machineId}.event")
	
	// Create a new machine to demonstrate EventBus
	def, _ := statemachine.NewBuilder("simple-machine").
		InitialState("idle").
		State("idle").
			On("start", "running").Done().
			Done().
		State("running").
			On("stop", "idle").Done().
			Done().
		Build()
	
	// Create with EventBus (would need actual EventBus instance in real scenario)
	sm, _ := statemachine.NewStateMachine(def)
	sm.Start(ctx)
	
	log.Printf("Simple machine created: %s (state: %s)", sm.ID(), sm.CurrentState())
	log.Println("In a real scenario, you can send events via EventBus:")
	log.Printf("  eventBus.Send('statemachine.%s.event', {name: 'start'})", sm.ID())
}
