# Fluxor State Machine

A comprehensive finite state machine implementation for Fluxor, featuring event-driven transitions via EventBus, declarative state definitions, guards, actions, and persistence.

## Features

- **Declarative State Definitions** - Define state machines in JSON or programmatically
- **Event-Driven Transitions** - State transitions triggered via EventBus
- **Guards** - Conditional logic to control transitions
- **Actions** - Execute code on state enter/exit and transitions
- **Hierarchical States** - Support for nested/composite states (future)
- **Persistence** - Save and restore state machine instances
- **Observable** - Listen to state changes in real-time
- **HTTP API** - RESTful API for state machine management
- **Type-Safe** - Strongly typed with Go generics support

## Table of Contents

1. [Quick Start](#quick-start)
2. [Core Concepts](#core-concepts)
3. [Building State Machines](#building-state-machines)
4. [Guards and Actions](#guards-and-actions)
5. [Persistence](#persistence)
6. [HTTP API](#http-api)
7. [EventBus Integration](#eventbus-integration)
8. [Examples](#examples)

---

## Quick Start

### 1. Create a State Machine

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/fluxor"
    "github.com/fluxorio/fluxor/pkg/statemachine"
)

func main() {
    app, _ := fluxor.NewMainVerticle("")

    // Deploy state machine verticle
    smVerticle := statemachine.NewVerticle(&statemachine.VerticleConfig{
        HTTPAddr: ":8082",
    })
    
    app.DeployVerticle(smVerticle)

    // Build state machine
    machine := statemachine.NewStateMachineBuilder("order", "Order FSM").
        InitialState("created").
        AddState("created", "Order Created").
            AddTransition("process", "processing").Done().
            Done().
        AddState("processing", "Processing").
            AddTransition("complete", "completed").Done().
            Done().
        AddState("completed", "Completed").Final(true).Done().
        Build()

    smVerticle.Engine().RegisterMachine(machine)
    
    app.Start()
}
```

### 2. Create an Instance and Send Events

```bash
# Create instance
curl -X POST http://localhost:8082/machines/order/instances \
  -H "Content-Type: application/json" \
  -d '{"initialData": {"orderId": "123", "amount": 100}}'

# Send event
curl -X POST http://localhost:8082/instances/{instanceId}/events \
  -H "Content-Type: application/json" \
  -d '{"name": "process", "data": {}}'

# Get instance state
curl http://localhost:8082/instances/{instanceId}
```

---

## Core Concepts

### State Machine Definition

A state machine consists of:

- **States** - The possible states the machine can be in
- **Transitions** - Rules for moving between states
- **Initial State** - The starting state
- **Final States** - Terminal states where the machine stops
- **Guards** - Conditions that must be true for a transition
- **Actions** - Code executed during transitions or state changes

### State Types

| Type | Description |
|------|-------------|
| `normal` | Standard state |
| `initial` | Starting state (automatically entered) |
| `final` | Terminal state (machine stops) |
| `parallel` | Multiple active substates (future) |

### Execution Context

Each state machine instance has an execution context containing:

- **Current State** - The active state
- **Data** - Instance-specific data
- **Variables** - User-defined variables
- **History** - Transition history
- **Status** - Running, completed, failed, or suspended

---

## Building State Machines

### Programmatic Builder API

The fluent builder API makes it easy to construct state machines:

```go
machine := statemachine.NewStateMachineBuilder("order-fsm", "Order State Machine").
    Description("Order processing workflow").
    Version("1.0.0").
    InitialState("created").
    
    // Define states
    AddState("created", "Order Created").
        Description("Initial state").
        AddTransition("validate", "validating").
            Guard("amountPositive").
            Action("validateOrder").
            Priority(1).
            Done().
        AddTransition("reject", "rejected").Done().
        Done().
    
    AddState("validating", "Validating Order").
        OnEnterAction(statemachine.FunctionAction("validateOrder")).
        AddTransition("process", "processing").Done().
        AddTransition("invalid", "rejected").Done().
        Done().
    
    AddState("processing", "Processing Order").
        OnEnterAction(statemachine.FunctionAction("processOrder")).
        AddTransition("complete", "completed").Done().
        AddTransition("fail", "failed").Done().
        Done().
    
    AddState("completed", "Completed").
        Final(true).
        OnEnterAction(statemachine.FunctionAction("completeOrder")).
        Done().
    
    AddState("rejected", "Rejected").Final(true).Done().
    AddState("failed", "Failed").Final(true).Done().
    
    Build()
```

### JSON Definition

You can also define state machines in JSON:

```json
{
  "id": "order-fsm",
  "name": "Order State Machine",
  "description": "Order processing workflow",
  "version": "1.0.0",
  "initialState": "created",
  "states": [
    {
      "id": "created",
      "name": "Order Created",
      "transitions": [
        {
          "event": "validate",
          "target": "validating",
          "guard": "amountPositive",
          "actions": [
            {
              "type": "function",
              "name": "validateOrder"
            }
          ],
          "priority": 1
        }
      ]
    },
    {
      "id": "validating",
      "name": "Validating Order",
      "transitions": [
        {
          "event": "process",
          "target": "processing"
        },
        {
          "event": "invalid",
          "target": "rejected"
        }
      ]
    },
    {
      "id": "processing",
      "name": "Processing Order",
      "onEnter": [
        {
          "type": "function",
          "name": "processOrder"
        }
      ],
      "transitions": [
        {
          "event": "complete",
          "target": "completed"
        },
        {
          "event": "fail",
          "target": "failed"
        }
      ]
    },
    {
      "id": "completed",
      "name": "Completed",
      "type": "final"
    },
    {
      "id": "rejected",
      "name": "Rejected",
      "type": "final"
    },
    {
      "id": "failed",
      "name": "Failed",
      "type": "final"
    }
  ]
}
```

---

## Guards and Actions

### Guards

Guards are conditions that determine if a transition can occur:

```go
// Register guard
smVerticle.RegisterGuard("amountPositive", func(
    ctx context.Context, 
    event *statemachine.Event, 
    execCtx *statemachine.ExecutionContext,
) bool {
    if event.Data == nil {
        return false
    }
    amount, ok := event.Data["amount"].(float64)
    return ok && amount > 0
})

// Use guard in transition
AddTransition("validate", "validating").
    Guard("amountPositive").
    Done()
```

#### Built-in Guard Helpers

```go
// Check if field > value
guard := statemachine.AmountGreaterThanGuard("amount", 100)

// Check if field exists
guard := statemachine.HasFieldGuard("orderId")

// Check if field equals value
guard := statemachine.EqualsGuard("status", "pending")
```

### Actions

Actions are executed during state transitions or state enter/exit:

```go
// Register action
smVerticle.RegisterAction("validateOrder", func(
    ctx context.Context,
    event *statemachine.Event,
    execCtx *statemachine.ExecutionContext,
) error {
    fmt.Printf("Validating order: %v\n", event.Data)
    execCtx.Variables["validated"] = true
    return nil
})

// Use action on state enter
AddState("validating", "Validating").
    OnEnterAction(statemachine.FunctionAction("validateOrder")).
    Done()

// Use action on transition
AddTransition("process", "processing").
    Action("processOrder").
    Done()
```

#### Action Types

| Type | Description | Config |
|------|-------------|--------|
| `function` | Execute registered function | `name`: function name |
| `eventbus` | Publish to EventBus | `address`: event address |
| `set` | Set variables | `values`: map of key-value |

#### Built-in Action Helpers

```go
// Publish to EventBus
action := statemachine.PublishAction("orders.processed")

// Send to EventBus
action := statemachine.SendAction("notifications.send")

// Set variable
action := statemachine.SetVariableAction("processed", true)

// Execute function
action := statemachine.FunctionAction("processOrder")
```

---

## Persistence

State machine instances can be persisted for recovery:

### Memory Persistence (Testing)

```go
persistence := statemachine.NewMemoryPersistenceProvider()

smVerticle := statemachine.NewVerticle(&statemachine.VerticleConfig{
    HTTPAddr:    ":8082",
    Persistence: persistence,
})
```

### File Persistence (Production)

```go
persistence, err := statemachine.NewFilePersistenceProvider("./fsm-data")
if err != nil {
    log.Fatal(err)
}

smVerticle := statemachine.NewVerticle(&statemachine.VerticleConfig{
    HTTPAddr:    ":8082",
    Persistence: persistence,
})
```

### Manual Save/Restore

```go
// Save instance
err := engine.SaveInstance(instanceID)

// Restore instance
err := engine.RestoreInstance(instanceID)
```

---

## HTTP API

The state machine verticle exposes a RESTful API:

### Machine Management

```bash
# Register machine
POST /machines
Content-Type: application/json
{
  "id": "order-fsm",
  "name": "Order State Machine",
  "initialState": "created",
  "states": [...]
}

# List machines
GET /machines

# Get machine definition
GET /machines/{id}
```

### Instance Management

```bash
# Create instance
POST /machines/{machineId}/instances
Content-Type: application/json
{
  "initialData": {
    "orderId": "123",
    "amount": 100
  }
}

# List instances
GET /machines/{machineId}/instances

# Get instance
GET /instances/{instanceId}

# Get instance history
GET /instances/{instanceId}/history
```

### Event Handling

```bash
# Send event to instance
POST /instances/{instanceId}/events
Content-Type: application/json
{
  "name": "process",
  "data": {
    "note": "Processing order"
  }
}
```

---

## EventBus Integration

State machines integrate seamlessly with Fluxor's EventBus:

### EventBus Addresses

| Address | Purpose |
|---------|---------|
| `statemachine.{machineId}.create` | Create new instance |
| `statemachine.{machineId}.event` | Send event to instance |
| `statemachine.{machineId}.query` | Query instance state |

### Creating Instance via EventBus

```go
// Request-reply pattern
msg, err := eventBus.Request(
    "statemachine.order-fsm.create",
    map[string]interface{}{
        "initialData": map[string]interface{}{
            "orderId": "123",
            "amount":  100,
        },
    },
    5*time.Second,
)

// Response contains instanceId
response := msg.Body().(map[string]interface{})
instanceID := response["instanceId"].(string)
```

### Sending Events via EventBus

```go
msg, err := eventBus.Request(
    "statemachine.order-fsm.event",
    map[string]interface{}{
        "instanceId": instanceID,
        "event":      "process",
        "data":       map[string]interface{}{},
    },
    5*time.Second,
)
```

### State Change Notifications

Listen to state changes in real-time:

```go
smVerticle.AddStateChangeListener(func(
    ctx context.Context,
    instanceID string,
    fromState string,
    toState string,
    event *statemachine.Event,
) {
    fmt.Printf("State changed: %s -> %s (event: %s)\n", 
        fromState, toState, event.Name)
    
    // Publish to EventBus for other components
    eventBus.Publish("statemachine.statechange", map[string]interface{}{
        "instanceId": instanceID,
        "from":       fromState,
        "to":         toState,
        "event":      event.Name,
    })
})
```

---

## Examples

### Example 1: Order Processing

See [examples/statemachine-demo](../../examples/statemachine-demo/) for a complete order processing state machine with:

- Multiple states and transitions
- Guards for validation
- Actions for processing
- State change listeners
- HTTP API integration
- Automatic progression through states

### Example 2: User Onboarding

```go
machine := statemachine.NewStateMachineBuilder("onboarding", "User Onboarding").
    InitialState("registered").
    
    AddState("registered", "Registered").
        AddTransition("verify_email", "email_verified").
            Guard("hasEmailToken").
            Action("sendVerificationEmail").
            Done().
        Done().
    
    AddState("email_verified", "Email Verified").
        AddTransition("complete_profile", "profile_completed").
            Guard("hasRequiredFields").
            Done().
        Done().
    
    AddState("profile_completed", "Profile Completed").
        AddTransition("activate", "active").Done().
        Done().
    
    AddState("active", "Active User").Final(true).Done().
    
    Build()
```

### Example 3: Payment Processing

```go
machine := statemachine.NewStateMachineBuilder("payment", "Payment FSM").
    InitialState("pending").
    
    AddState("pending", "Payment Pending").
        AddTransition("authorize", "authorized").
            Guard("validCard").
            Action("authorizePayment").
            Done().
        AddTransition("fail", "failed").Done().
        Done().
    
    AddState("authorized", "Payment Authorized").
        AddTransition("capture", "captured").
            Action("capturePayment").
            Done().
        AddTransition("void", "voided").Done().
        Done().
    
    AddState("captured", "Payment Captured").
        AddTransition("refund", "refunded").
            Guard("refundable").
            Action("processRefund").
            Done().
        Done().
    
    AddState("failed", "Payment Failed").Final(true).Done().
    AddState("voided", "Payment Voided").Final(true).Done().
    AddState("refunded", "Payment Refunded").Final(true).Done().
    
    Build()
```

---

## Architecture

### State Machine Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application                              │
└──────────────────┬──────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                   State Machine Verticle                         │
│  • HTTP API Server (optional)                                   │
│  • Guard/Action Registry                                        │
│  • State Change Listeners                                       │
└──────────────────┬──────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────────┐
│                   State Machine Engine                           │
│  • Machine Registry                                              │
│  • Instance Management                                           │
│  • Transition Processing                                         │
│  • Guard Evaluation                                              │
│  • Action Execution                                              │
└──────────────────┬──────────────────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │                     │
        ▼                     ▼
┌──────────────┐    ┌──────────────────┐
│   EventBus   │    │   Persistence    │
│              │    │    Provider      │
│ • Publish    │    │ • Save           │
│ • Send       │    │ • Load           │
│ • Request    │    │ • Delete         │
└──────────────┘    └──────────────────┘
```

### State Transition Flow

```
Event Received
    ↓
Find Current State
    ↓
Find Matching Transition (by priority)
    ↓
Evaluate Guard (if present)
    ├─→ false: Reject Transition
    └─→ true: Continue
         ↓
Execute OnExit Actions (current state)
    ↓
Execute Transition Actions
    ↓
Update Current State
    ↓
Record Transition in History
    ↓
Execute OnEnter Actions (target state)
    ↓
Notify State Change Listeners
    ↓
Persist Instance (if configured)
    ↓
Check if Final State → Update Status
```

---

## Best Practices

### 1. State Naming

- Use clear, descriptive state names
- Follow consistent naming conventions (e.g., `verb_past_tense`)
- Example: `created`, `processing`, `completed`

### 2. Guard Design

- Keep guards simple and focused
- Avoid side effects in guards
- Return boolean explicitly
- Use descriptive guard names

### 3. Action Design

- Actions should be idempotent when possible
- Handle errors gracefully
- Log important actions
- Keep actions focused on single responsibility

### 4. Transition Priority

- Use priority when multiple transitions share the same event
- Higher priority = evaluated first
- Default priority is 0

### 5. Error Handling

- Always return errors from actions
- Use state change listeners for monitoring
- Log transition failures
- Consider adding error states for recovery

### 6. Testing

- Test each state independently
- Test all transition paths
- Test guard conditions
- Test error scenarios
- Use memory persistence for tests

---

## Advanced Topics

### Custom Persistence Provider

Implement the `PersistenceProvider` interface:

```go
type CustomPersistence struct {
    db *sql.DB
}

func (p *CustomPersistence) Save(instanceID string, execCtx *statemachine.ExecutionContext) error {
    // Save to database
    return nil
}

func (p *CustomPersistence) Load(instanceID string) (*statemachine.ExecutionContext, error) {
    // Load from database
    return nil, nil
}

func (p *CustomPersistence) Delete(instanceID string) error {
    // Delete from database
    return nil
}

func (p *CustomPersistence) List(machineID string) ([]string, error) {
    // List from database
    return nil, nil
}
```

### State Handlers

Implement custom state handlers:

```go
type ProcessingStateHandler struct{}

func (h *ProcessingStateHandler) OnEnter(
    ctx context.Context,
    event *statemachine.Event,
    execCtx *statemachine.ExecutionContext,
) error {
    fmt.Println("Entering processing state")
    return nil
}

func (h *ProcessingStateHandler) OnExit(
    ctx context.Context,
    event *statemachine.Event,
    execCtx *statemachine.ExecutionContext,
) error {
    fmt.Println("Exiting processing state")
    return nil
}

// Register handler
smVerticle.RegisterStateHandler("processing", &ProcessingStateHandler{})
```

---

## Troubleshooting

### Issue: Transition Not Occurring

**Possible Causes:**
- Guard returning false
- Event name mismatch
- Instance not in expected state
- Instance in final state

**Solution:**
- Check guard logic
- Verify event names match exactly
- Check current state via API
- Inspect instance status

### Issue: Action Failing

**Possible Causes:**
- Action not registered
- Action returning error
- Context data missing

**Solution:**
- Verify action is registered with correct name
- Check action logs for errors
- Ensure required data is in event or context

### Issue: Instance Not Persisting

**Possible Causes:**
- Persistence provider not configured
- File system permissions
- Serialization error

**Solution:**
- Verify persistence provider is set
- Check file system permissions
- Review logs for serialization errors

---

## Performance Considerations

1. **State Machine Size** - Keep number of states reasonable (< 50 states)
2. **Transition Count** - Limit transitions per state (< 20)
3. **Action Duration** - Keep actions fast (< 100ms)
4. **Guard Complexity** - Simple boolean checks only
5. **History Size** - Consider truncating history for long-running instances
6. **Persistence** - Use async persistence for high throughput

---

## Future Enhancements

- [ ] Hierarchical/nested states
- [ ] Parallel states (multiple active states)
- [ ] State machine composition
- [ ] Time-based transitions
- [ ] Conditional branching with multiple guards
- [ ] State machine visualization
- [ ] Metrics and monitoring
- [ ] State machine templates

---

## See Also

- [examples/statemachine-demo](../../examples/statemachine-demo/) - Complete working example
- [Fluxor Architecture](../../ARCHITECTURE.md) - Overall Fluxor architecture
- [EventBus Documentation](../core/eventbus.go) - EventBus integration
- [Workflow Engine](../workflow/README.md) - Related workflow engine
