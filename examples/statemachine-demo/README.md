# State Machine Demo

This example demonstrates the Fluxor State Machine in action.

## What This Example Shows

1. **State Machine Definition**: Building a complex order processing state machine
2. **Verticle Integration**: Deploying state machine as a Fluxor Verticle
3. **Entry/Exit Handlers**: Executing logic on state changes
4. **Guards**: Conditional transitions based on event data
5. **Actions**: Side effects during transitions
6. **HTTP API**: Managing state machines via REST API
7. **Event-Driven**: Integration with Fluxor's EventBus

## State Machine Diagram

```
                    ┌─────────┐
                    │ pending │ (initial)
                    └────┬────┘
                         │
                    ┌────┴────┐
                    │         │
              approve│         │reject
                    │         │
              ┌─────▼───┐ ┌───▼────┐
              │approved │ │rejected│ (final)
              └────┬────┘ └────────┘
                   │
              process│
                   │
              ┌────▼────────┐
              │ processing  │
              └────┬────┬───┘
                   │    │
            complete│    │fail
                   │    │
          ┌────────▼─┐ ┌▼──────┐
          │completed │ │ failed│ (final)
          └──────────┘ └───────┘
          (final)
```

## Running the Example

```bash
cd examples/statemachine-demo
go run main.go
```

The example will:
1. Start a state machine verticle with HTTP API on `:8082`
2. Register an order processing state machine definition
3. Create a sample machine and demonstrate transitions
4. Show the transition history

## HTTP API Examples

### Create a Machine

```bash
curl -X POST http://localhost:8082/machines \
  -H "Content-Type: application/json" \
  -d '{
    "definitionId": "order-processing",
    "machineId": "order-456",
    "initialContext": {
      "customerID": "customer-789"
    }
  }'
```

Response:
```json
{
  "machineId": "order-456",
  "state": "pending"
}
```

### Send an Event

```bash
curl -X POST http://localhost:8082/machines/order-456/events \
  -H "Content-Type: application/json" \
  -d '{
    "name": "approve",
    "data": {
      "orderId": "order-456",
      "approvedBy": "admin"
    }
  }'
```

Response:
```json
{
  "success": true,
  "currentState": "approved"
}
```

### Get Machine Status

```bash
curl http://localhost:8082/machines/order-456
```

Response:
```json
{
  "id": "order-456",
  "currentState": "approved",
  "definition": "order-processing",
  "history": [
    {
      "from": "pending",
      "to": "approved",
      "event": "approve",
      "timestamp": "2024-01-01T12:00:00Z",
      "duration": "1.5s"
    }
  ]
}
```

### List All Machines

```bash
curl http://localhost:8082/machines
```

### Reset a Machine

```bash
curl -X POST http://localhost:8082/machines/order-456/reset
```

## Sample Output

```
2024/01/01 12:00:00 State machine verticle deployed with HTTP API on :8082
2024/01/01 12:00:00 Example: Create a machine and send events via HTTP
2024/01/01 12:00:00   POST http://localhost:8082/machines {"definitionId": "order-processing"}
2024/01/01 12:00:00   POST http://localhost:8082/machines/{id}/events {"name": "approve", "data": {"orderId": "123"}}

========================================
Demonstrating State Machine
========================================

2024/01/01 12:00:02 Created state machine: 3f5a2b1c-...
2024/01/01 12:00:02 Current state: pending

2024/01/01 12:00:02 📋 Order is pending review...
2024/01/01 12:00:03 Sending 'approve' event...
2024/01/01 12:00:03 ✓ Order review completed
2024/01/01 12:00:03 ✓ Approving order 123...
2024/01/01 12:00:03 ✓ Order approved! Ready for processing.
2024/01/01 12:00:03 Current state: approved

2024/01/01 12:00:04 Sending 'process' event...
2024/01/01 12:00:04 ⚙️  Starting order processing...
2024/01/01 12:00:04 ⚙️  Processing order...
2024/01/01 12:00:04 Current state: processing

2024/01/01 12:00:05 Sending 'complete' event...
2024/01/01 12:00:05 📦 Preparing shipment...
2024/01/01 12:00:05 ✅ Order completed successfully!
2024/01/01 12:00:05 Current state: completed

========================================
State Transition History:
========================================
1. pending -> approved (event: approve, duration: 1.5s)
2. approved -> processing (event: process, duration: 1.2s)
3. processing -> completed (event: complete, duration: 800ms)

========================================
```

## Code Walkthrough

### Building the State Machine

```go
def, err := statemachine.NewBuilder("order-processing").
    Name("Order Processing").
    InitialState("pending").
    
    State("pending").
        Entry(func(ctx context.Context, event statemachine.Event) error {
            log.Println("📋 Order is pending review...")
            return nil
        }).
        On("approve", "approved").
            Guard(statemachine.DataFieldExists("orderId")).
            Action(approveOrder).
            Done().
        Done().
    
    // ... more states
    
    Build()
```

### Deploying as Verticle

```go
app, _ := fluxor.NewMainVerticle("config.json")

smVerticle := statemachine.NewStateMachineVerticle(&statemachine.StateMachineVerticleConfig{
    HTTPAddr: ":8082",
})

app.DeployVerticle(smVerticle)
smVerticle.RegisterDefinition(def)
```

### Creating and Using Machines

```go
machine, _ := verticle.CreateMachine(ctx, "order-processing", "order-123", nil)

machine.Send(ctx, statemachine.Event{
    Name: "approve",
    Data: map[string]interface{}{
        "orderId": "order-123",
    },
    Timestamp: time.Now(),
})

fmt.Printf("Current state: %s\n", machine.CurrentState())
```

## Key Concepts Demonstrated

### 1. Fluent Builder API

The example shows how to use the fluent builder to create complex state machines with minimal boilerplate.

### 2. Guards and Actions

Guards validate conditions before transitions, while actions perform side effects during transitions.

### 3. Entry/Exit Handlers

Handlers execute when entering or exiting states, perfect for logging and cleanup.

### 4. Event-Driven Architecture

State machines integrate with Fluxor's EventBus for distributed systems.

### 5. HTTP Management

The verticle provides a full REST API for managing state machines remotely.

## Extending This Example

### Add Persistence

```go
persistence, _ := statemachine.NewFilePersistenceAdapter("./state-data")

smVerticle := statemachine.NewStateMachineVerticle(&statemachine.StateMachineVerticleConfig{
    HTTPAddr:    ":8082",
    Persistence: persistence,
})
```

### Add Observers

```go
logger := core.NewDefaultLogger()
observer := statemachine.NewLoggingObserver(logger)

machine, _ := statemachine.NewStateMachine(def,
    statemachine.WithObserver(observer),
)
```

### Add Metrics

```go
metricsObserver := statemachine.NewMetricsObserver()

machine, _ := statemachine.NewStateMachine(def,
    statemachine.WithObserver(metricsObserver),
)

// Later...
metrics := metricsObserver.GetMetrics()
log.Printf("Total transitions: %v", metrics["transitions"])
```

## See Also

- [pkg/statemachine/README.md](../../pkg/statemachine/README.md) - Full documentation
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - Fluxor architecture
- [docs/PRIMARY_PATTERN.md](../../docs/PRIMARY_PATTERN.md) - Fluxor patterns
