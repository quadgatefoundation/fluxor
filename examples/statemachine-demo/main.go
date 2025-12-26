// Example: State Machine using EventBus
//
// This demonstrates how to create and execute state machines using Fluxor.
// The example shows an order processing state machine with multiple states and transitions.
//
// Run: go run ./examples/statemachine-demo
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/statemachine"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatal(err)
	}

	// Deploy state machine verticle with HTTP API
	smVerticle := statemachine.NewVerticle(&statemachine.VerticleConfig{
		HTTPAddr: ":8082", // State machine management API
	})

	// Register custom guards
	smVerticle.RegisterGuard("amountPositive", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) bool {
		if event.Data == nil {
			return false
		}
		if amount, ok := event.Data["amount"].(float64); ok {
			return amount > 0
		}
		return false
	})

	smVerticle.RegisterGuard("amountGreaterThan100", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) bool {
		// Check in execution context data
		if amount, ok := execCtx.Data["finalAmount"].(float64); ok {
			return amount > 100
		}
		return false
	})

	// Register custom actions
	smVerticle.RegisterAction("validateOrder", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) error {
		fmt.Printf("✅ Validating order: %v\n", event.Data)
		execCtx.Variables["validated"] = true
		return nil
	})

	smVerticle.RegisterAction("processOrder", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) error {
		fmt.Printf("⚙️  Processing order: %s\n", execCtx.InstanceID)
		execCtx.Variables["processedAt"] = time.Now()
		
		// Calculate discount
		if amount, ok := execCtx.Data["amount"].(float64); ok {
			var discount float64
			if amount > 100 {
				discount = amount * 0.1 // 10% discount
			} else if amount > 50 {
				discount = amount * 0.05 // 5% discount
			}
			execCtx.Data["discount"] = discount
			execCtx.Data["finalAmount"] = amount - discount
			fmt.Printf("💰 Amount: %.2f, Discount: %.2f, Final: %.2f\n", amount, discount, amount-discount)
		}
		return nil
	})

	smVerticle.RegisterAction("completeOrder", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) error {
		fmt.Printf("✨ Order completed: %s (tier: %v)\n", execCtx.InstanceID, execCtx.Variables["tier"])
		return nil
	})

	smVerticle.RegisterAction("failOrder", func(ctx context.Context, event *statemachine.Event, execCtx *statemachine.ExecutionContext) error {
		fmt.Printf("❌ Order failed: %s (reason: %v)\n", execCtx.InstanceID, event.Data["reason"])
		return nil
	})

	// Add state change listener
	smVerticle.AddStateChangeListener(func(ctx context.Context, instanceID, from, to string, event *statemachine.Event) {
		if from == "" {
			fmt.Printf("🎬 State Machine Started: %s -> %s\n", instanceID, to)
		} else {
			fmt.Printf("🔄 State Transition: %s -> %s (event: %s)\n", from, to, event.Name)
		}
	})

	app.DeployVerticle(smVerticle)

	// Build the order processing state machine
	machine := statemachine.NewStateMachineBuilder("order-fsm", "Order Processing State Machine").
		Description("A comprehensive order processing workflow with validation, processing, and completion").
		Version("1.0.0").
		InitialState("created").
		AddState("created", "Order Created").
		Description("Initial state when order is created").
		AddTransition("validate", "validating").
		Guard("amountPositive").
		Action("validateOrder").
		Priority(1).
		Done().
		AddTransition("reject", "rejected").
		Priority(0).
		Done().
		Done().
		AddState("validating", "Validating Order").
		Description("Validating order data").
		AddTransition("process", "processing").Done().
		AddTransition("invalid", "rejected").Done().
		Done().
		AddState("processing", "Processing Order").
		Description("Processing the order").
		OnEnterAction(statemachine.FunctionAction("processOrder")).
		AddTransition("complete", "completed").Done().
		AddTransition("fail", "failed").
		Action("failOrder").
		Done().
		Done().
		AddState("completed", "Order Completed").
		Description("Order successfully completed").
		Final(true).
		OnEnterAction(statemachine.FunctionAction("completeOrder")).
		Done().
		AddState("rejected", "Order Rejected").
		Description("Order was rejected").
		Final(true).
		Done().
		AddState("failed", "Order Failed").
		Description("Order processing failed").
		Final(true).
		Done().
		Build()

	// Register the state machine
	if err := smVerticle.Engine().RegisterMachine(machine); err != nil {
		log.Fatal(err)
	}

	// Deploy API gateway for easy testing
	app.DeployVerticle(NewApiGateway(smVerticle))

	fmt.Println("\n🚀 State Machine Demo Running")
	fmt.Println("   API Gateway:     http://localhost:8080")
	fmt.Println("   State Machine:   http://localhost:8082")
	fmt.Println("")
	fmt.Println("📋 Try these commands:")
	fmt.Println("")
	fmt.Println("   # Create an order (triggers state machine)")
	fmt.Println(`   curl -X POST http://localhost:8080/api/orders -H "Content-Type: application/json" -d '{"orderId":"ORD-001","amount":150}'`)
	fmt.Println("")
	fmt.Println("   # List all state machine instances")
	fmt.Println("   curl http://localhost:8082/machines/order-fsm/instances")
	fmt.Println("")
	fmt.Println("   # Get instance details")
	fmt.Println("   curl http://localhost:8082/instances/{instanceId}")
	fmt.Println("")
	fmt.Println("   # Get instance history")
	fmt.Println("   curl http://localhost:8082/instances/{instanceId}/history")
	fmt.Println("")
	fmt.Println("   # Send event to instance")
	fmt.Println(`   curl -X POST http://localhost:8082/instances/{instanceId}/events -H "Content-Type: application/json" -d '{"name":"process","data":{}}'`)
	fmt.Println("")

	app.Start()
}

// ApiGateway verticle provides REST API for testing
type ApiGateway struct {
	smVerticle *statemachine.Verticle
	server     *web.FastHTTPServer
}

func NewApiGateway(smVerticle *statemachine.Verticle) *ApiGateway {
	return &ApiGateway{smVerticle: smVerticle}
}

func (v *ApiGateway) Start(ctx core.FluxorContext) error {
	config := web.DefaultFastHTTPServerConfig(":8080")
	v.server = web.NewFastHTTPServer(ctx.Vertx(), config)
	router := v.server.FastRouter()

	// Health check
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{"status": "ok"})
	})

	// Create order (creates state machine instance)
	router.POSTFast("/api/orders", func(c *web.FastRequestContext) error {
		var orderData map[string]interface{}
		if err := c.BindJSON(&orderData); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		// Create state machine instance
		instanceID, err := v.smVerticle.Engine().CreateInstance(c.Context(), "order-fsm", orderData)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Automatically trigger validation
		time.Sleep(100 * time.Millisecond) // Brief delay to ensure state is ready

		validateEvent := &statemachine.Event{
			Name:      "validate",
			Data:      orderData,
			Timestamp: time.Now(),
		}
		if err := v.smVerticle.Engine().SendEvent(c.Context(), instanceID, validateEvent); err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Auto-progress to processing
		time.Sleep(100 * time.Millisecond)
		processEvent := &statemachine.Event{
			Name:      "process",
			Data:      map[string]interface{}{},
			Timestamp: time.Now(),
		}
		if err := v.smVerticle.Engine().SendEvent(c.Context(), instanceID, processEvent); err != nil {
			// May fail if already transitioned, that's ok
		}

		// Auto-complete
		time.Sleep(100 * time.Millisecond)
		completeEvent := &statemachine.Event{
			Name:      "complete",
			Data:      map[string]interface{}{},
			Timestamp: time.Now(),
		}
		if err := v.smVerticle.Engine().SendEvent(c.Context(), instanceID, completeEvent); err != nil {
			// May fail if already transitioned, that's ok
		}

		// Get final state
		instance, err := v.smVerticle.Engine().GetInstance(instanceID)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(201, map[string]interface{}{
			"instanceId":   instanceID,
			"currentState": instance.CurrentState,
			"status":       instance.Status,
			"data":         instance.Data,
			"variables":    instance.Variables,
		})
	})

	// Get order status
	router.GETFast("/api/orders/:id", func(c *web.FastRequestContext) error {
		instanceID := c.Params["id"]

		instance, err := v.smVerticle.Engine().GetInstance(instanceID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": "order not found"})
		}

		return c.JSON(200, map[string]interface{}{
			"instanceId":   instanceID,
			"currentState": instance.CurrentState,
			"status":       instance.Status,
			"data":         instance.Data,
			"variables":    instance.Variables,
			"history":      instance.History,
		})
	})

	// List all orders
	router.GETFast("/api/orders", func(c *web.FastRequestContext) error {
		instances, err := v.smVerticle.Engine().ListInstances("order-fsm")
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(200, map[string]interface{}{
			"orders": instances,
			"count":  len(instances),
		})
	})

	go v.server.Start()
	return nil
}

func (v *ApiGateway) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
