# Fluxor FSM (Finite State Machine)

A lightweight, reactive Finite State Machine for Go, inspired by [Spring State Machine](https://spring.io/projects/spring-statemachine) (Java) and [Stateless](https://github.com/dotnet-state-machine/stateless) (.NET).

Built on top of the Fluxor runtime, utilizing `FutureT` for asynchronous, non-blocking state transitions.

## Features

- **Fluent API**: Builder pattern for easy configuration.
- **Reactive**: `Fire()` executes asynchronously and returns a `FutureT`.
- **Guards**: Conditional transitions.
- **Actions**: Entry, Exit, and Transition actions.
- **Internal Transitions**: Execute actions without changing state or triggering Entry/Exit.
- **Thread-Safe**: Safe for concurrent use.

## Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/fluxorio/fluxor/pkg/fsm"
)

func main() {
    // 1. Create FSM
    machine := fsm.New("order-123", "Created")

    // 2. Configure States
    machine.Configure("Created").
        Permit("Pay", "Paid").
        OnExit(func(ctx context.Context, t fsm.TransitionContext) error {
            fmt.Println("Processing payment...")
            return nil
        })

    machine.Configure("Paid").
        Permit("Ship", "Shipped").
        OnEntry(func(ctx context.Context, t fsm.TransitionContext) error {
            fmt.Println("Payment received!")
            return nil
        })

    machine.Configure("Shipped").
        Ignore("Pay") // Ignore extra payments

    // 3. Fire Events
    ctx := context.Background()
    
    // Returns a FutureT[State]
    future := machine.Fire(ctx, "Pay", paymentData)
    
    newState, err := future.Await(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Current state: %s\n", newState)
}
```

## Transition Types

- **External** (`Permit`, `PermitIf`): Transitions from Source to Target. Exits Source, Enters Target.
- **Internal** (`InternalTransition`): Executes action but stays in Source. Does NOT Exit Source or Enter Target.

## Integration with Fluxor

The FSM is designed to run inside a Verticle or as a standalone component. Since `Fire` returns a `FutureT`, it integrates seamlessly with Fluxor's async patterns.
