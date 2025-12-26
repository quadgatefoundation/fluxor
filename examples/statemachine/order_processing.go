package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/statemachine"
)

// OrderProcessingExample demonstrates a state machine for order processing.
//
// States: pending → payment_processing → confirmed → shipped → delivered
//         pending → cancelled (on cancel event)
//         payment_processing → failed (on payment failure)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Fluxor app using MainVerticle
	app, err := fluxor.NewMainVerticle("config.json")
	if err != nil {
		// Use default config if file not found
		log.Printf("Config file not found, using defaults: %v", err)
		app, err = fluxor.NewMainVerticleWithOptions("", fluxor.MainVerticleOptions{})
		if err != nil {
			log.Fatal(err)
		}
	}

	// Create state machine verticle
	smVerticle := statemachine.NewStateMachineVerticle()

	// Deploy the verticle
	if _, err := app.Vertx().DeployVerticle(smVerticle); err != nil {
		log.Fatalf("Failed to deploy state machine verticle: %v", err)
	}

	// Create and register the order processing state machine
	orderSM := createOrderProcessingStateMachine()
	if err := smVerticle.RegisterStateMachine(orderSM, nil); err != nil {
		log.Fatalf("Failed to register state machine: %v", err)
	}

	log.Println("Order Processing State Machine deployed successfully")

	// Create a state machine client
	client := statemachine.NewStateMachineClient(app.Vertx().EventBus())

	// Example: Create an order instance
	orderID, err := client.CreateInstance(ctx, "order-processing", map[string]interface{}{
		"orderId":      "ORD-12345",
		"customerId":   "CUST-789",
		"totalAmount":  99.99,
		"items":        []string{"ITEM-1", "ITEM-2"},
		"orderDate":    time.Now().Format(time.RFC3339),
	})
	if err != nil {
		log.Fatalf("Failed to create order instance: %v", err)
	}

	log.Printf("Created order instance: %s", orderID)

	// Simulate order processing workflow
	go simulateOrderProcessing(ctx, client, orderID)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Order Processing State Machine is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")
	if err := app.Stop(); err != nil {
		log.Printf("Error stopping app: %v", err)
	}
}

func createOrderProcessingStateMachine() *statemachine.StateMachineDefinition {
	builder := statemachine.NewBuilder("order-processing", "Order Processing")
	builder.WithDescription("State machine for processing customer orders").
		WithVersion("1.0").
		WithInitialState("pending")

	// Define states
	builder.AddStates(
		// Pending: Order created, waiting for payment
		statemachine.NewState("pending", "Pending").
			WithDescription("Order is pending payment").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Order is pending payment", ctx.MachineID)
				return nil
			}).
			Build(),

		// Payment Processing: Processing payment
		statemachine.NewState("payment_processing", "Payment Processing").
			WithDescription("Payment is being processed").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Processing payment...", ctx.MachineID)
				// Simulate payment gateway call
				ctx.Data["paymentStartTime"] = time.Now()
				return nil
			}).
			OnExit(func(ctx *statemachine.StateContext) error {
				duration := time.Since(ctx.Data["paymentStartTime"].(time.Time))
				log.Printf("[Order %s] Payment processing took %v", ctx.MachineID, duration)
				return nil
			}).
			Build(),

		// Confirmed: Payment successful, order confirmed
		statemachine.NewState("confirmed", "Confirmed").
			WithDescription("Order confirmed, ready for shipping").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Order confirmed! Payment successful.", ctx.MachineID)
				ctx.Data["confirmedAt"] = time.Now()
				// Notify customer
				return nil
			}).
			Build(),

		// Shipped: Order has been shipped
		statemachine.NewState("shipped", "Shipped").
			WithDescription("Order has been shipped").
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Order shipped!", ctx.MachineID)
				ctx.Data["shippedAt"] = time.Now()
				// Generate tracking number
				ctx.Data["trackingNumber"] = fmt.Sprintf("TRK-%s", ctx.MachineID[:8])
				log.Printf("[Order %s] Tracking number: %s", ctx.MachineID, ctx.Data["trackingNumber"])
				return nil
			}).
			Build(),

		// Delivered: Order successfully delivered (final state)
		statemachine.NewState("delivered", "Delivered").
			WithDescription("Order successfully delivered").
			AsFinal().
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Order delivered successfully! 🎉", ctx.MachineID)
				ctx.Data["deliveredAt"] = time.Now()
				// Send satisfaction survey
				return nil
			}).
			Build(),

		// Cancelled: Order cancelled (final state)
		statemachine.NewState("cancelled", "Cancelled").
			WithDescription("Order was cancelled").
			AsFinal().
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Order cancelled", ctx.MachineID)
				ctx.Data["cancelledAt"] = time.Now()
				// Process refund if needed
				if ctx.Data["paymentProcessed"] == true {
					log.Printf("[Order %s] Processing refund...", ctx.MachineID)
				}
				return nil
			}).
			Build(),

		// Failed: Payment failed (final state)
		statemachine.NewState("failed", "Failed").
			WithDescription("Payment failed").
			AsFinal().
			OnEnter(func(ctx *statemachine.StateContext) error {
				log.Printf("[Order %s] Payment failed: %v", ctx.MachineID, ctx.Data["failureReason"])
				// Notify customer of failure
				return nil
			}).
			Build(),
	)

	// Define transitions
	builder.AddTransitions(
		// pending → payment_processing
		statemachine.NewTransition("process-payment", "pending", "payment_processing", "process_payment").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				log.Printf("[Order %s] Initiating payment processing", ctx.MachineID)
				return nil
			}).
			Build(),

		// payment_processing → confirmed (on payment success)
		statemachine.NewTransition("payment-success", "payment_processing", "confirmed", "payment_success").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				ctx.Data["paymentProcessed"] = true
				ctx.Data["paymentId"] = event.Data["paymentId"]
				ctx.Data["paymentMethod"] = event.Data["paymentMethod"]
				log.Printf("[Order %s] Payment successful (ID: %v)", ctx.MachineID, event.Data["paymentId"])
				return nil
			}).
			Build(),

		// payment_processing → failed (on payment failure)
		statemachine.NewTransition("payment-failed", "payment_processing", "failed", "payment_failed").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				ctx.Data["failureReason"] = event.Data["reason"]
				log.Printf("[Order %s] Payment failed: %v", ctx.MachineID, event.Data["reason"])
				return nil
			}).
			Build(),

		// confirmed → shipped
		statemachine.NewTransition("ship-order", "confirmed", "shipped", "ship").
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				// Check if warehouse has confirmed availability
				warehouseReady, ok := event.Data["warehouseReady"].(bool)
				if !ok || !warehouseReady {
					log.Printf("[Order %s] Warehouse not ready for shipping", ctx.MachineID)
					return false, nil
				}
				return true, nil
			}).
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				ctx.Data["carrier"] = event.Data["carrier"]
				log.Printf("[Order %s] Shipping via %v", ctx.MachineID, event.Data["carrier"])
				return nil
			}).
			Build(),

		// shipped → delivered
		statemachine.NewTransition("deliver-order", "shipped", "delivered", "deliver").
			WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
				ctx.Data["signedBy"] = event.Data["signedBy"]
				log.Printf("[Order %s] Delivered and signed by %v", ctx.MachineID, event.Data["signedBy"])
				return nil
			}).
			Build(),

		// pending → cancelled (can cancel before payment)
		statemachine.NewTransition("cancel-pending", "pending", "cancelled", "cancel").
			WithPriority(10).
			Build(),

		// payment_processing → cancelled (can cancel during payment)
		statemachine.NewTransition("cancel-processing", "payment_processing", "cancelled", "cancel").
			WithPriority(10).
			WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
				// Only allow cancel if payment not yet completed
				if ctx.Data["paymentProcessed"] == true {
					return false, nil
				}
				return true, nil
			}).
			Build(),

		// confirmed → cancelled (can cancel after confirmation but before shipping)
		statemachine.NewTransition("cancel-confirmed", "confirmed", "cancelled", "cancel").
			WithPriority(10).
			Build(),
	)

	definition, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build state machine: %v", err)
	}

	return definition
}

func simulateOrderProcessing(ctx context.Context, client *statemachine.StateMachineClient, orderID string) {
	// Wait a bit before starting
	time.Sleep(2 * time.Second)

	// Query initial state
	state, err := client.QueryInstance(ctx, "order-processing", orderID)
	if err != nil {
		log.Printf("Failed to query order: %v", err)
		return
	}
	log.Printf("Initial state: %v", state["currentState"])

	// Step 1: Process payment
	time.Sleep(1 * time.Second)
	log.Println("\n=== Sending process_payment event ===")
	success, err := client.SendEvent(ctx, "order-processing", orderID, "process_payment", nil)
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Event sent successfully: %v", success)

	// Step 2: Simulate payment success
	time.Sleep(2 * time.Second)
	log.Println("\n=== Sending payment_success event ===")
	success, err = client.SendEvent(ctx, "order-processing", orderID, "payment_success", map[string]interface{}{
		"paymentId":     "PAY-" + orderID[:8],
		"paymentMethod": "credit_card",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Event sent successfully: %v", success)

	// Step 3: Ship order
	time.Sleep(2 * time.Second)
	log.Println("\n=== Sending ship event ===")
	success, err = client.SendEvent(ctx, "order-processing", orderID, "ship", map[string]interface{}{
		"warehouseReady": true,
		"carrier":        "FastShip Express",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Event sent successfully: %v", success)

	// Step 4: Deliver order
	time.Sleep(2 * time.Second)
	log.Println("\n=== Sending deliver event ===")
	success, err = client.SendEvent(ctx, "order-processing", orderID, "deliver", map[string]interface{}{
		"signedBy": "John Doe",
	})
	if err != nil {
		log.Printf("Failed to send event: %v", err)
		return
	}
	log.Printf("Event sent successfully: %v", success)

	// Query final state
	time.Sleep(1 * time.Second)
	state, err = client.QueryInstance(ctx, "order-processing", orderID)
	if err != nil {
		log.Printf("Failed to query order: %v", err)
		return
	}
	log.Printf("\n=== Final State ===")
	log.Printf("Current state: %v", state["currentState"])
	log.Printf("Status: %v", state["status"])
	log.Printf("Data: %v", state["data"])
}
