# Fluxor State Machine

A powerful, event-driven state machine implementation for Fluxor that integrates seamlessly with the EventBus, Verticles, and reactive patterns.

## Features

- **Event-Driven**: State transitions triggered by events via EventBus
- **Verticle Integration**: Deploy as a Verticle for lifecycle management
- **Guards & Actions**: Conditional transitions and side effects
- **Entry/Exit Handlers**: Execute logic when entering or exiting states
- **Async Support**: Send events asynchronously with Future/Promise pattern
- **Persistence**: Optional state persistence (memory, file, EventBus)
- **Observability**: Built-in observers for logging, metrics, and EventBus notifications
- **Fluent Builder API**: Intuitive fluent API for building state machines
- **HTTP API**: Optional REST API for managing state machines
- **Type-Safe**: Strongly typed with comprehensive error handling

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/fluxorio/fluxor/pkg/statemachine"
)

func main() {
    // Build a state machine definition
    def, err := statemachine.NewBuilder("order-machine").
        Name("Order Processing").
        InitialState("pending").
        
        State("pending").
            Entry(func(ctx context.Context, event statemachine.Event) error {
                log.Println("Order is pending review")
                return nil
            }).
            On("approve", "approved").Done().
            On("reject", "rejected").Done().
            Done().
        
        State("approved").
            On("ship", "shipped").Done().
            Done().
        
        State("shipped").
            Final(true).
            Done().
        
        State("rejected").
            Final(true).
            Done().
        
        Build()

    if err != nil {
        log.Fatal(err)
    }

    // Create a state machine instance
    sm, err := statemachine.NewStateMachine(def)
    if err != nil {
        log.Fatal(err)
    }

    // Start the state machine
    ctx := context.Background()
    if err := sm.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Send events to transition states
    sm.Send(ctx, statemachine.Event{
        Name:      "approve",
        Timestamp: time.Now(),
    })

    log.Printf("Current state: %s", sm.CurrentState())
}
```

### With Fluxor Verticle

```go
package main

import (
    "log"

    "github.com/fluxorio/fluxor/pkg/fluxor"
    "github.com/fluxorio/fluxor/pkg/statemachine"
)

func main() {
    // Create Fluxor app
    app, err := fluxor.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    // Create state machine verticle with HTTP API
    smVerticle := statemachine.NewStateMachineVerticle(&statemachine.StateMachineVerticleConfig{
        HTTPAddr: ":8082",
    })

    // Deploy the verticle
    app.DeployVerticle(smVerticle)

    // Register state machine definition
    def := buildStateMachineDefinition()
    smVerticle.RegisterDefinition(def)

    // Start the app
    app.Start()
}
```

## Core Concepts

### States

States represent the possible conditions of your system. Each state can have:

- **Entry handler**: Executed when entering the state
- **Exit handler**: Executed when leaving the state
- **Transitions**: Events that can move to other states
- **Final flag**: Marks a state as terminal

```go
State("processing").
    Entry(func(ctx context.Context, event statemachine.Event) error {
        log.Println("Started processing")
        return nil
    }).
    Exit(func(ctx context.Context, event statemachine.Event) error {
        log.Println("Finished processing")
        return nil
    }).
    On("complete", "completed").Done().
    On("error", "failed").Done().
    Final(false).
    Done()
```

### Transitions

Transitions define how the state machine moves between states:

```go
On("approve", "approved").
    Guard(func(ctx context.Context, event statemachine.Event) (bool, error) {
        // Only allow if orderID exists
        _, ok := event.Data["orderId"]
        return ok, nil
    }).
    Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
        // Execute approval logic
        log.Printf("Approving order: %v", event.Data["orderId"])
        return nil
    }).
    Priority(10).
    Timeout(5 * time.Second).
    Done()
```

### Guards

Guards are conditions that must be satisfied for a transition to occur:

```go
// Built-in guards
On("proceed", "next").
    Guard(statemachine.DataFieldExists("requiredField")).
    Done()

// Custom guards
On("proceed", "next").
    Guard(func(ctx context.Context, event statemachine.Event) (bool, error) {
        amount, ok := event.Data["amount"].(int)
        return ok && amount > 100, nil
    }).
    Done()

// Composite guards
On("proceed", "next").
    Guard(statemachine.AndGuard(
        statemachine.DataFieldExists("field1"),
        statemachine.DataFieldExists("field2"),
    )).
    Done()
```

### Actions

Actions are executed during transitions:

```go
On("process", "processing").
    Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
        // Perform processing
        log.Printf("Transitioning from %s to %s", from, to)
        return nil
    }).
    Done()

// Chain multiple actions
On("complete", "completed").
    Action(statemachine.ChainActions(
        notifyCustomer,
        updateInventory,
        sendAnalytics,
    )).
    Done()
```

### Events

Events trigger state transitions:

```go
event := statemachine.Event{
    Name: "approve",
    Data: map[string]interface{}{
        "orderId":    "order-123",
        "approvedBy": "admin",
        "amount":     1500,
    },
    Timestamp: time.Now(),
    RequestID: "req-456",
}

sm.Send(ctx, event)
```

## Advanced Features

### Async Transitions

Send events asynchronously and handle results with Futures:

```go
future := sm.SendAsync(ctx, statemachine.Event{
    Name: "process",
    Timestamp: time.Now(),
})

// Wait for completion
err := future.Await(ctx)

// Or register callback
future.OnComplete(func(err error) {
    if err != nil {
        log.Printf("Transition failed: %v", err)
    } else {
        log.Println("Transition succeeded")
    }
})
```

### State Persistence

Persist state across restarts:

```go
// Memory persistence (for testing)
persistence := statemachine.NewMemoryPersistenceAdapter()

// File persistence
persistence, _ := statemachine.NewFilePersistenceAdapter("./state-data")

// Create machine with persistence
sm, err := statemachine.NewStateMachine(def,
    statemachine.WithPersistence(persistence),
)
```

### Observability

Monitor state transitions with observers:

```go
// Logging observer
logger := core.NewDefaultLogger()
loggingObserver := statemachine.NewLoggingObserver(logger)

// Metrics observer
metricsObserver := statemachine.NewMetricsObserver()

// EventBus observer
eventBusObserver := statemachine.NewEventBusObserver(eventBus, "statemachine.events")

// Create machine with observers
sm, err := statemachine.NewStateMachine(def,
    statemachine.WithObserver(loggingObserver),
    statemachine.WithObserver(metricsObserver),
    statemachine.WithObserver(eventBusObserver),
)

// Get metrics
metrics := metricsObserver.GetMetrics()
log.Printf("Transitions: %v", metrics["transitions"])
```

### EventBus Integration

Integrate with Fluxor's EventBus for distributed state machines:

```go
// Create machine with EventBus
sm, err := statemachine.NewStateMachine(def,
    statemachine.WithEventBus(eventBus),
)

// Send events via EventBus
eventBus.Send(fmt.Sprintf("statemachine.%s.event", sm.ID()), statemachine.Event{
    Name: "approve",
    Timestamp: time.Now(),
})

// Subscribe to state changes
eventBus.Consumer(fmt.Sprintf("statemachine.%s.transition", sm.ID())).
    Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var change statemachine.StateChangeEvent
        // Handle state change
        return nil
    })
```

### HTTP API

When deployed as a Verticle with HTTP enabled:

```bash
# Register a definition
POST /definitions
{
  "id": "order-machine",
  "name": "Order Processing",
  "initialState": "pending",
  "states": {...}
}

# Create a machine instance
POST /machines
{
  "definitionId": "order-machine",
  "machineId": "order-123",
  "initialContext": {"customerID": "cust-456"}
}

# Send an event
POST /machines/order-123/events
{
  "name": "approve",
  "data": {"orderId": "123", "approvedBy": "admin"}
}

# Get machine status
GET /machines/order-123

# Reset machine
POST /machines/order-123/reset

# List all machines
GET /machines

# List definitions
GET /definitions
```

## Builder API Reference

### StateMachine Builder

```go
builder := statemachine.NewBuilder("machine-id")
```

#### Methods

- `Name(string)` - Set machine name
- `Description(string)` - Set description
- `Version(string)` - Set version
- `InitialState(string)` - Set initial state (required)
- `Context(map[string]interface{})` - Set shared context
- `State(string)` - Add a state (returns StateBuilder)
- `Build()` - Build and validate definition
- `BuildAndCreate(...Option)` - Build and create instance

### State Builder

```go
State("stateName")
```

#### Methods

- `Final(bool)` - Mark as final state
- `Entry(Handler)` - Set entry handler
- `Exit(Handler)` - Set exit handler
- `Metadata(string, interface{})` - Add metadata
- `On(event, targetState)` - Add transition (returns TransitionBuilder)
- `Done()` - Finish state and return to machine builder

### Transition Builder

```go
On("eventName", "targetState")
```

#### Methods

- `Guard(Guard)` - Add guard condition
- `Action(Action)` - Add transition action
- `Priority(int)` - Set priority (higher = evaluated first)
- `Timeout(time.Duration)` - Set action timeout
- `Metadata(string, interface{})` - Add metadata
- `Done()` - Finish transition and return to state builder
- `OnDone(event, target)` - Shortcut to add another transition

## Guard Functions

Built-in guard functions:

```go
statemachine.AlwaysAllow()
statemachine.NeverAllow()
statemachine.DataFieldEquals("field", value)
statemachine.DataFieldExists("field")
statemachine.AndGuard(guard1, guard2, ...)
statemachine.OrGuard(guard1, guard2, ...)
statemachine.NotGuard(guard)
```

## Action Functions

Built-in action functions:

```go
statemachine.NoOpAction()
statemachine.LogAction(func(msg string) { log.Println(msg) })
statemachine.ChainActions(action1, action2, ...)
```

## Options

Configuration options when creating a state machine:

```go
sm, err := statemachine.NewStateMachine(def,
    statemachine.WithID("custom-id"),
    statemachine.WithEventBus(eventBus),
    statemachine.WithLogger(logger),
    statemachine.WithPersistence(adapter),
    statemachine.WithObserver(observer),
    statemachine.WithInitialContext(map[string]interface{}{
        "key": "value",
    }),
)
```

## Error Handling

The state machine uses typed errors:

```go
err := sm.Send(ctx, event)
if err != nil {
    if smErr, ok := err.(*statemachine.StateMachineError); ok {
        switch smErr.Code {
        case statemachine.ErrorCodeInvalidTransition:
            log.Println("No valid transition for this event")
        case statemachine.ErrorCodeGuardRejected:
            log.Println("Guard condition not satisfied")
        case statemachine.ErrorCodeActionFailed:
            log.Println("Transition action failed")
        case statemachine.ErrorCodeTimeout:
            log.Println("Action timed out")
        }
    }
}
```

## Examples

### Order Processing

Complete example of order lifecycle management:

```go
def, _ := statemachine.NewBuilder("order-processing").
    InitialState("pending").
    State("pending").
        On("approve", "approved").
            Guard(statemachine.DataFieldExists("orderId")).
            Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
                orderID := event.Data["orderId"]
                return approveOrder(orderID)
            }).
            Done().
        On("reject", "rejected").Done().
        Done().
    State("approved").
        On("ship", "shipped").Done().
        Done().
    State("shipped").
        Final(true).
        Done().
    State("rejected").
        Final(true).
        Done().
    Build()
```

### Traffic Light

Simple traffic light state machine:

```go
def, _ := statemachine.NewBuilder("traffic-light").
    InitialState("red").
    State("red").
        On("timer", "green").
            Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
                time.Sleep(3 * time.Second)
                return nil
            }).
            Done().
        Done().
    State("green").
        On("timer", "yellow").Done().
        Done().
    State("yellow").
        On("timer", "red").Done().
        Done().
    Build()
```

### User Authentication

Authentication flow state machine:

```go
def, _ := statemachine.NewBuilder("auth").
    InitialState("unauthenticated").
    State("unauthenticated").
        On("login", "authenticating").Done().
        Done().
    State("authenticating").
        On("success", "authenticated").
            Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
                return createSession(event.Data["userId"])
            }).
            Done().
        On("failure", "unauthenticated").Done().
        On("mfa_required", "mfa_pending").Done().
        Done().
    State("mfa_pending").
        On("verify", "authenticated").Done().
        On("cancel", "unauthenticated").Done().
        Done().
    State("authenticated").
        On("logout", "unauthenticated").Done().
        Done().
    Build()
```

## Best Practices

### 1. Use Guards for Validation

```go
// Good: Guard validates before transition
On("submit", "processing").
    Guard(statemachine.AndGuard(
        statemachine.DataFieldExists("formData"),
        validateFormData,
    )).
    Done()

// Avoid: Validation in action (state already changed)
On("submit", "processing").
    Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
        if !isValid(event.Data) {
            return errors.New("invalid data") // Too late!
        }
        return nil
    }).
    Done()
```

### 2. Keep Actions Idempotent

Actions may be retried, so make them idempotent:

```go
Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
    // Idempotent: safe to call multiple times
    return database.UpsertRecord(event.Data["id"], event.Data)
})
```

### 3. Use Timeouts for External Operations

```go
On("process", "processing").
    Action(func(ctx context.Context, from, to string, event statemachine.Event) error {
        return callExternalAPI(ctx, event.Data)
    }).
    Timeout(5 * time.Second).
    Done()
```

### 4. Leverage Observers for Side Effects

```go
// Separate concerns: use observers for logging, metrics, notifications
observer := statemachine.NewChainObserver(
    loggingObserver,
    metricsObserver,
    eventBusObserver,
)
```

### 5. Persist State for Critical Workflows

```go
// For critical workflows, always enable persistence
sm, _ := statemachine.NewStateMachine(def,
    statemachine.WithPersistence(adapter),
)
```

## Testing

### Unit Tests

```go
func TestOrderApproval(t *testing.T) {
    def := buildOrderMachine()
    sm, _ := statemachine.NewStateMachine(def)
    
    ctx := context.Background()
    sm.Start(ctx)
    
    err := sm.Send(ctx, statemachine.Event{
        Name: "approve",
        Data: map[string]interface{}{"orderId": "123"},
        Timestamp: time.Now(),
    })
    
    if err != nil {
        t.Fatalf("Failed to approve: %v", err)
    }
    
    if sm.CurrentState() != "approved" {
        t.Errorf("Expected state 'approved', got '%s'", sm.CurrentState())
    }
}
```

### Integration Tests

```go
func TestStateMachineVerticle(t *testing.T) {
    app, _ := fluxor.NewMainVerticle("")
    verticle := statemachine.NewStateMachineVerticle(&statemachine.StateMachineVerticleConfig{
        HTTPAddr: ":9999",
    })
    
    app.DeployVerticle(verticle)
    
    // Test via HTTP API
    resp, _ := http.Post("http://localhost:9999/machines", "application/json", ...)
    // ... assertions
}
```

## Architecture

The state machine integrates with Fluxor's architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    Fluxor Application                        │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         StateMachineVerticle                         │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐     │   │
│  │  │ Definition │  │ Definition │  │ Definition │     │   │
│  │  │  Registry  │  │  Registry  │  │  Registry  │     │   │
│  │  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘     │   │
│  │        │                │                │           │   │
│  │  ┌─────▼────────────────▼────────────────▼──────┐    │   │
│  │  │         State Machine Instances             │    │   │
│  │  │  ┌─────────┐  ┌─────────┐  ┌─────────┐      │    │   │
│  │  │  │Machine 1│  │Machine 2│  │Machine 3│      │    │   │
│  │  │  └────┬────┘  └────┬────┘  └────┬────┘      │    │   │
│  │  └───────┼────────────┼────────────┼───────────┘    │   │
│  └──────────┼────────────┼────────────┼────────────────┘   │
│             │            │            │                     │
│  ┌──────────▼────────────▼────────────▼──────────────────┐ │
│  │                 EventBus                               │ │
│  │  • statemachine.{id}.event                             │ │
│  │  • statemachine.{id}.transition                        │ │
│  │  • statemachine.register                               │ │
│  │  • statemachine.create                                 │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

## Performance

The state machine is designed for high performance:

- **Lock-free reads**: Current state can be read without locks
- **Single transition lock**: Ensures one transition at a time per instance
- **Non-blocking**: EventBus integration is non-blocking
- **Minimal allocations**: Reuses structures where possible

## See Also

- [examples/statemachine-demo](../../examples/statemachine-demo/) - Complete working example
- [ARCHITECTURE.md](../../ARCHITECTURE.md) - Fluxor architecture overview
- [docs/PRIMARY_PATTERN.md](../../docs/PRIMARY_PATTERN.md) - Fluxor patterns
- [pkg/workflow/README.md](../workflow/README.md) - Workflow engine (complementary)

## License

MIT
