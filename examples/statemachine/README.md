# State Machine Examples

This directory contains examples demonstrating the Fluxor State Machine implementation.

## Examples

### 1. Order Processing (`order_processing.go`)

A complete order processing workflow demonstrating:

- **States**: `pending → payment_processing → confirmed → shipped → delivered`
- **Features**:
  - Payment processing with success/failure handling
  - Order confirmation and shipping
  - Delivery tracking with signature
  - Cancellation support at multiple stages
  - State entry/exit actions
  - Transition guards and actions
  - Data flow through states

#### State Diagram

```
┌─────────┐  process_payment  ┌────────────────────┐
│ pending ├──────────────────►│ payment_processing │
└────┬────┘                   └──────┬─────────────┘
     │                               │
     │ cancel                        │ payment_success
     │                               │
     │                        ┌──────▼────────┐
     │                        │   confirmed   │
     │                        └──────┬────────┘
     │                               │
     │                               │ ship
     │                               │
     │                        ┌──────▼────┐  deliver  ┌───────────┐
     │                        │  shipped  ├──────────►│ delivered │
     │                        └───────────┘           └───────────┘
     │
     │                        ┌───────────┐
     └───────────────────────►│ cancelled │
                              └───────────┘
```

#### Running

```bash
cd examples/statemachine
go run order_processing.go
```

#### Output Example

```
2024/12/26 Order Processing State Machine deployed successfully
2024/12/26 Created order instance: abc-123-def-456
2024/12/26 [Order abc-123] Order is pending payment

=== Sending process_payment event ===
2024/12/26 [Order abc-123] Initiating payment processing
2024/12/26 [Order abc-123] Processing payment...

=== Sending payment_success event ===
2024/12/26 [Order abc-123] Payment successful (ID: PAY-abc-123)
2024/12/26 [Order abc-123] Order confirmed! Payment successful.

=== Sending ship event ===
2024/12/26 [Order abc-123] Shipping via FastShip Express
2024/12/26 [Order abc-123] Order shipped!
2024/12/26 [Order abc-123] Tracking number: TRK-abc-123

=== Sending deliver event ===
2024/12/26 [Order abc-123] Delivered and signed by John Doe
2024/12/26 [Order abc-123] Order delivered successfully! 🎉
```

### 2. Approval Workflow (`approval_workflow.go`)

A multi-level approval system demonstrating:

- **States**: `draft → pending_l1 → pending_l2 → pending_l3 → approved`
- **Features**:
  - Three-level approval hierarchy (Manager → Director → Executive)
  - Conditional routing based on approval amount
  - Rejection handling at any level
  - Revision request workflow
  - Priority-based transition selection
  - Guard conditions for role-based authorization
  - Comprehensive approval tracking

#### State Diagram

```
┌───────┐   submit   ┌─────────────┐   approve   ┌─────────────┐
│ draft ├───────────►│ pending_l1  ├────────────►│ pending_l2  │
└───┬───┘            └──────┬──────┘             └──────┬──────┘
    │                       │                           │
    │                       │ reject                    │ approve (high value)
    │                       │                           │
    │                ┌──────▼─────┐             ┌──────▼──────┐
    │                │  rejected  │             │ pending_l3  │
    │                └────────────┘             └──────┬──────┘
    │                                                  │
    │                                                  │ approve
    │                                                  │
    │                                           ┌──────▼────┐
    │                                           │ approved  │
    │                                           └───────────┘
    │                                           
    │                ┌────────────────────┐
    └───────────────►│ revision_requested │
                     └─────────┬──────────┘
                               │
                               │ resubmit
                               │
                               └──────────┐
                                         ▼
```

#### Running

```bash
cd examples/statemachine
go run approval_workflow.go
```

#### Output Example

```
2024/12/26 Approval Workflow State Machine deployed successfully
2024/12/26 Created document instance: DOC-2024-001
2024/12/26 [Doc DOC-2024] Document in draft state

=== Submitting document for approval ===
2024/12/26 [Doc DOC-2024] Submitted for Level 1 approval (Manager)

=== Manager approving (Level 1) ===
2024/12/26 [Doc DOC-2024] ✓ Level 1 approved by manager@example.com
2024/12/26 [Doc DOC-2024] Submitted for Level 2 approval (Director)

=== Director approving (Level 2) ===
2024/12/26 [Doc DOC-2024] ✓ Level 2 approved by director@example.com
2024/12/26 [Doc DOC-2024] High value item, escalating to Level 3
2024/12/26 [Doc DOC-2024] Submitted for Level 3 approval (Executive)

=== Executive approving (Level 3) ===
2024/12/26 [Doc DOC-2024] ✓ Level 3 approved by ceo@example.com
2024/12/26 [Doc DOC-2024] ✅ Document fully approved!
```

## Common Patterns Demonstrated

### 1. State Entry/Exit Actions

```go
statemachine.NewState("processing", "Processing").
    OnEnter(func(ctx *statemachine.StateContext) error {
        log.Printf("Starting processing for %s", ctx.MachineID)
        ctx.Data["startTime"] = time.Now()
        return nil
    }).
    OnExit(func(ctx *statemachine.StateContext) error {
        duration := time.Since(ctx.Data["startTime"].(time.Time))
        log.Printf("Processing completed in %v", duration)
        return nil
    }).
    Build()
```

### 2. Guard Conditions

```go
statemachine.NewTransition("approve", "pending", "approved", "approve").
    WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
        // Check authorization
        role, ok := event.Data["approver_role"].(string)
        if !ok || role != "manager" {
            return false, nil
        }
        
        // Check business rules
        amount := ctx.Data["amount"].(float64)
        return amount < 10000, nil
    }).
    Build()
```

### 3. Transition Actions

```go
statemachine.NewTransition("payment-success", "pending", "confirmed", "payment_success").
    WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
        // Record payment details
        ctx.Data["paymentId"] = event.Data["paymentId"]
        ctx.Data["paymentMethod"] = event.Data["paymentMethod"]
        ctx.Data["paymentTime"] = time.Now()
        
        // Publish event
        ctx.FluxorContext.EventBus().Publish("payment.confirmed", map[string]interface{}{
            "orderId":   ctx.Data["orderId"],
            "paymentId": event.Data["paymentId"],
        })
        
        return nil
    }).
    Build()
```

### 4. Priority-Based Selection

```go
// High priority - evaluated first
builder.AddTransition(
    statemachine.NewTransition("route-urgent", "pending", "express", "route").
        WithPriority(10).
        WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
            priority := ctx.Data["priority"].(string)
            return priority == "urgent", nil
        }).
        Build(),
)

// Lower priority - evaluated if first guard fails
builder.AddTransition(
    statemachine.NewTransition("route-normal", "pending", "standard", "route").
        WithPriority(5).
        Build(),
)
```

### 5. EventBus Integration

```go
// Create client
client := statemachine.NewStateMachineClient(eventBus)

// Create instance via EventBus
instanceID, err := client.CreateInstance(ctx, "order-processing", initialData)

// Send event via EventBus
success, err := client.SendEvent(ctx, "order-processing", instanceID, "approve", eventData)

// Query state via EventBus
state, err := client.QueryInstance(ctx, "order-processing", instanceID)
```

## Building and Running

### Prerequisites

- Go 1.21 or later
- Fluxor framework

### Build

```bash
# Build all examples
go build ./examples/statemachine/...

# Build specific example
go build -o order_processor examples/statemachine/order_processing.go
go build -o approval_workflow examples/statemachine/approval_workflow.go
```

### Run

```bash
# Run directly
go run examples/statemachine/order_processing.go
go run examples/statemachine/approval_workflow.go

# Or run built binaries
./order_processor
./approval_workflow
```

## Creating Your Own State Machine

1. **Define States**:
   ```go
   states := []*statemachine.State{
       statemachine.SimpleState("initial", "Initial State"),
       statemachine.SimpleState("active", "Active State"),
       statemachine.FinalState("done", "Done State"),
   }
   ```

2. **Define Transitions**:
   ```go
   transitions := []*statemachine.Transition{
       statemachine.SimpleTransition("start", "initial", "active", "start"),
       statemachine.SimpleTransition("finish", "active", "done", "finish"),
   }
   ```

3. **Build Definition**:
   ```go
   builder := statemachine.NewBuilder("my-sm", "My State Machine")
   builder.WithInitialState("initial")
   builder.AddStates(states...)
   builder.AddTransitions(transitions...)
   definition, _ := builder.Build()
   ```

4. **Deploy and Use**:
   ```go
   smVerticle := statemachine.NewStateMachineVerticle()
   vertx.DeployVerticle(smVerticle)
   smVerticle.RegisterStateMachine(definition, nil)
   
   client := statemachine.NewStateMachineClient(eventBus)
   instanceID, _ := client.CreateInstance(ctx, "my-sm", initialData)
   client.SendEvent(ctx, "my-sm", instanceID, "start", nil)
   ```

## Next Steps

- Read the [State Machine API Documentation](../../pkg/statemachine/README.md)
- Explore the [test files](../../pkg/statemachine/engine_test.go) for more patterns
- Check the [Fluxor documentation](../../README.md) for framework features

## Troubleshooting

### State machine not responding to events

- Ensure the verticle is deployed: `vertx.DeployVerticle(smVerticle)`
- Check EventBus is enabled in config: `config.EnableEventBus = true`
- Verify event name matches transition event exactly
- Check guard conditions aren't blocking the transition

### Instance not found

- Verify instance was created successfully
- Check instance ID is correct
- Enable persistence if instances need to survive restarts

### Transitions not executing

- Check current state matches transition's `From` state
- Verify guard conditions return `true`
- Check transition priority if multiple transitions exist
- Ensure no errors in OnExit/Action/OnEnter handlers

## Support

For questions and issues:
- File an issue on the Fluxor GitHub repository
- Check the [Fluxor documentation](../../README.md)
- Review the [test files](../../pkg/statemachine/) for examples
