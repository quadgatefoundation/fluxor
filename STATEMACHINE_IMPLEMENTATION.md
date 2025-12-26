# State Machine Implementation for Fluxor

## Summary

A comprehensive finite state machine (FSM) implementation has been built on top of the Fluxor framework, featuring event-driven transitions via EventBus, declarative state definitions, guards, actions, and persistence.

## What Was Built

### 1. Core Package: `pkg/statemachine/`

The state machine package includes the following components:

#### `types.go` - Core Types and Interfaces
- **StateMachineDefinition** - JSON-serializable state machine definition
- **StateDefinition** - Individual state with transitions, actions, and metadata
- **TransitionDefinition** - State transitions with guards and actions
- **ExecutionContext** - Runtime context for state machine instances
- **Event** - Events that trigger state transitions
- **Guard/Action Functions** - Function signatures for conditional logic and side effects
- **Interfaces** - StateMachineEngine, StateMachineVerticle, PersistenceProvider

#### `engine.go` - State Machine Engine
- **Engine** - Core state machine execution engine
  - Machine registration and validation
  - Instance creation and lifecycle management
  - Event processing and state transitions
  - Guard evaluation with priority-based selection
  - Action execution (function, eventbus, set variables)
  - State change notifications
  - EventBus integration for distributed state machines
  - Persistence support

#### `verticle.go` - Deployable Verticle
- **Verticle** - Fluxor verticle for state machine deployment
  - HTTP API server (optional)
  - Guard/action registration
  - State handler registration
  - State change listeners
- **HTTP API Endpoints**:
  - `POST /machines` - Register state machine
  - `GET /machines` - List all machines
  - `POST /machines/:id/instances` - Create instance
  - `GET /machines/:id/instances` - List instances
  - `GET /instances/:id` - Get instance state
  - `POST /instances/:id/events` - Send event
  - `GET /instances/:id/history` - Get transition history

#### `builder.go` - Fluent Builder API
- **StateMachineBuilder** - Fluent API for building state machines
  - Chainable methods for states, transitions, guards, actions
  - Type-safe construction
  - Programmatic definition
- **Helper Functions**:
  - `PublishAction()`, `SendAction()`, `SetVariableAction()`
  - `AmountGreaterThanGuard()`, `HasFieldGuard()`, `EqualsGuard()`

#### `persistence.go` - Persistence Providers
- **MemoryPersistenceProvider** - In-memory persistence (testing)
- **FilePersistenceProvider** - File-based persistence (production)
- Support for custom persistence implementations

#### `engine_test.go` - Comprehensive Tests
- Basic state machine flow
- Guards with conditional transitions
- Actions with side effects
- Transition priority evaluation
- Persistence save/restore
- State change listeners

### 2. Example: `examples/statemachine-demo/`

A complete working example demonstrating:
- Order processing state machine
- Multiple states: created → validating → processing → completed/rejected/failed
- Guards for validation (amount checks)
- Actions for processing logic (validation, processing, completion)
- State change listeners for monitoring
- HTTP API integration
- Automatic state progression

### 3. Documentation: `pkg/statemachine/README.md`

Comprehensive documentation including:
- Quick start guide
- Core concepts explanation
- Building state machines (programmatic and JSON)
- Guards and actions
- Persistence options
- HTTP API reference
- EventBus integration
- Multiple examples
- Architecture diagrams
- Best practices
- Troubleshooting guide

## Key Features

### 1. Event-Driven Architecture
- State transitions triggered by events via EventBus
- Distributed state machines across services
- Asynchronous event processing
- Request-reply patterns for state queries

### 2. Declarative State Definitions
- JSON-serializable definitions
- Programmatic fluent builder API
- Clear state and transition modeling
- Metadata support

### 3. Guards (Conditions)
- Control when transitions can occur
- Priority-based evaluation
- Custom guard functions
- Built-in guard helpers

### 4. Actions (Side Effects)
- Execute on state enter/exit
- Execute during transitions
- Multiple action types:
  - Function actions (custom Go functions)
  - EventBus actions (publish/send)
  - Variable setting (state management)

### 5. Persistence
- Save/restore state machine instances
- Memory persistence for testing
- File-based persistence for production
- Custom persistence providers

### 6. Observable
- State change listeners
- Real-time notifications
- Integration with monitoring systems
- Event publishing on state changes

### 7. HTTP API
- RESTful API for management
- Machine registration
- Instance creation and querying
- Event sending
- History viewing

## Integration with Fluxor

The state machine integrates seamlessly with Fluxor's architecture:

### EventBus Integration
```
statemachine.{machineId}.create  → Create instance
statemachine.{machineId}.event   → Send event
statemachine.{machineId}.query   → Query state
```

### Verticle Pattern
- Deployable unit via `app.DeployVerticle(smVerticle)`
- Lifecycle management (Start/Stop)
- Configuration injection

### Context and Logger
- Uses `core.FluxorContext`
- Structured logging via `core.Logger`
- Request ID propagation

### Concurrency
- Thread-safe engine operations
- Mutex-protected state access
- Goroutine-safe listeners

## Usage Example

```go
// Create and deploy state machine verticle
smVerticle := statemachine.NewVerticle(&statemachine.VerticleConfig{
    HTTPAddr: ":8082",
})
app.DeployVerticle(smVerticle)

// Build state machine
machine := statemachine.NewStateMachineBuilder("order", "Order FSM").
    InitialState("created").
    AddState("created", "Order Created").
        AddTransition("process", "processing").
            Guard("amountPositive").
            Action("validateOrder").
            Done().
        Done().
    AddState("processing", "Processing").
        OnEnterAction(statemachine.FunctionAction("processOrder")).
        AddTransition("complete", "completed").Done().
        Done().
    AddState("completed", "Completed").Final(true).Done().
    Build()

smVerticle.Engine().RegisterMachine(machine)

// Create instance
instanceID, _ := smVerticle.Engine().CreateInstance(ctx, "order", data)

// Send event
event := &statemachine.Event{Name: "process", Data: map[string]interface{}{}}
smVerticle.Engine().SendEvent(ctx, instanceID, event)

// Get state
instance, _ := smVerticle.Engine().GetInstance(instanceID)
fmt.Println("Current State:", instance.CurrentState)
```

## Architecture

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

## State Transition Flow

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

## Testing

All tests pass successfully:
- ✅ Basic state machine flow
- ✅ Guards with conditional transitions
- ✅ Actions with side effects
- ✅ Transition priority evaluation
- ✅ Persistence save/restore
- ✅ State change listeners

## Files Created

1. `/workspace/pkg/statemachine/types.go` - Core types and interfaces
2. `/workspace/pkg/statemachine/engine.go` - State machine engine implementation
3. `/workspace/pkg/statemachine/verticle.go` - Deployable verticle with HTTP API
4. `/workspace/pkg/statemachine/builder.go` - Fluent builder API
5. `/workspace/pkg/statemachine/persistence.go` - Persistence providers
6. `/workspace/pkg/statemachine/engine_test.go` - Comprehensive tests
7. `/workspace/pkg/statemachine/README.md` - Complete documentation
8. `/workspace/examples/statemachine-demo/main.go` - Working example
9. `/workspace/examples/statemachine-demo/order-fsm.json` - JSON definition example
10. `/workspace/STATEMACHINE_IMPLEMENTATION.md` - This summary

## Running the Example

```bash
# Run the state machine demo
go run ./examples/statemachine-demo

# In another terminal, create an order
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -d '{"orderId":"ORD-001","amount":150}'

# List all instances
curl http://localhost:8082/machines/order-fsm/instances

# Get instance details
curl http://localhost:8082/instances/{instanceId}

# View transition history
curl http://localhost:8082/instances/{instanceId}/history
```

## Next Steps

Potential enhancements:
- [ ] Hierarchical/nested states
- [ ] Parallel states (multiple active states)
- [ ] State machine composition
- [ ] Time-based transitions
- [ ] State machine visualization (Graphviz/Mermaid)
- [ ] Metrics and monitoring integration
- [ ] State machine templates/library

## Conclusion

The state machine implementation is production-ready and fully integrated with Fluxor's architecture. It provides:

1. **Declarative** - Define state machines in JSON or programmatically
2. **Event-Driven** - Transitions via EventBus
3. **Observable** - Listen to state changes
4. **Persistent** - Save and restore instances
5. **Testable** - Comprehensive test coverage
6. **Documented** - Complete documentation and examples
7. **Scalable** - Distributed via EventBus
8. **Type-Safe** - Strong typing with Go

The implementation follows Fluxor's patterns (Verticles, EventBus, Context) and integrates seamlessly with the existing ecosystem.
