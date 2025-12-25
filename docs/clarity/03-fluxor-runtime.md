# Fluxor Runtime Deep Dive: MainVerticle, Future/Promise, Async Patterns

This document provides a detailed analysis of `pkg/fluxor/` focusing on:
1. MainVerticle bootstrap pattern
2. Future/Promise and FutureT/PromiseT (typed) patterns
3. Async combinators (All, Race, Then, Catch)
4. **Confusing spots → Suggested fixes**

---

## 1. MainVerticle Bootstrap Pattern

### Flow Diagram

```
NewMainVerticle(configPath)
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  1. Create root context with cancel                                         │
│     rootCtx, cancel := context.WithCancel(context.Background())             │
└─────────────────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  2. Load config from file (JSON/YAML)                                       │
│     config.Load(configPath, &cfg)                                           │
└─────────────────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  3. Create Vertx with optional custom EventBus                              │
│     core.NewVertxWithOptions(rootCtx, vopts)                                │
│     ├── Default: in-memory EventBus                                         │
│     └── Custom: opts.EventBusFactory (e.g., NATS)                           │
└─────────────────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  MainVerticle ready                                                         │
│  ├── .Vertx() → access underlying Vertx                                     │
│  ├── .Config() → access loaded config                                       │
│  ├── .DeployVerticle(v) → deploy with config injection                      │
│  ├── .Start() → block on SIGINT/SIGTERM                                     │
│  └── .Stop() → cancel context + close Vertx                                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Config Injection

```go
DeployVerticle(v)
         │
         ├── Is AsyncVerticle?
         │   ├── YES: wrap with configInjectedAsyncVerticle
         │   └── NO:  wrap with configInjectedVerticle
         │
         ▼
     vertx.DeployVerticle(wrapped)
         │
         ▼
     On Start/Stop:
         for k, val := range cfg {
             ctx.SetConfig(k, val)  // Inject all config keys
         }
         return inner.Start(ctx)   // Then call actual verticle
```

---

## 2. Future/Promise Patterns

### Type Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Untyped (interface{})              │  Typed (generics)                     │
│  ─────────────────────────────────  │  ─────────────────────────────────    │
│  Future (interface)                 │  FutureT[T] (struct)                  │
│  ├── Complete(interface{})          │  ├── Await(ctx) → (T, error)          │
│  ├── Fail(error)                    │  ├── OnSuccess(func(T))               │
│  ├── OnSuccess(func(interface{}))   │  └── OnFailure(func(error))           │
│  ├── OnFailure(func(error))         │                                       │
│  ├── Await(ctx) → (interface{}, err)│  PromiseT[T] (struct, embeds FutureT) │
│  ├── Then(fn) → Future              │  ├── Complete(T)                      │
│  ├── Catch(fn) → Future             │  └── Fail(error)                      │
│  └── Map(fn) → Future               │                                       │
│                                     │  Global functions:                    │
│  Promise (interface, extends Future)│  ├── Then[T,R](f, fn) → FutureT[R]    │
│  ├── TryComplete(interface{}) bool  │  ├── Catch[T](f, fn) → FutureT[T]     │
│  └── TryFail(error) bool            │  ├── Map[T,R](f, fn) → FutureT[R]     │
│                                     │  ├── All[T](ctx, ...f) → FutureT[[]T] │
│                                     │  └── Race[T](ctx, ...f) → FutureT[T]  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Usage Patterns

```go
// Pattern 1: Untyped Future/Promise (Vert.x style)
promise := fluxor.NewPromise()
go func() {
    result, err := doWork()
    if err != nil {
        promise.Fail(err)
    } else {
        promise.Complete(result)
    }
}()
result, err := promise.Await(ctx)

// Pattern 2: Typed FutureT/PromiseT (Go generics style)
promise := fluxor.NewPromiseT[User]()
promise.Complete(User{Name: "John"})
user, err := promise.Await(ctx) // user is User, not interface{}

// Pattern 3: Async EventBus request
future := fluxor.RequestAsync[UserResponse](eb, ctx, "user.get", req, 5*time.Second)
response, err := future.Await(ctx)

// Pattern 4: Combinators
f1 := fluxor.RequestAsync[int](eb, ctx, "calc.add", req1, timeout)
f2 := fluxor.RequestAsync[int](eb, ctx, "calc.mul", req2, timeout)
results := fluxor.All[int](ctx, f1, f2) // Wait for both
first := fluxor.Race[int](ctx, f1, f2)  // First to complete
```

---

## 3. Implementation Details

### Future State Machine

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              Future States                                │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────┐     Complete(v)     ┌────────────────────────┐             │
│  │ PENDING  │ ──────────────────► │ COMPLETED (Value: v)   │             │
│  └──────────┘                     └────────────────────────┘             │
│       │                                                                   │
│       │ Fail(err)                                                        │
│       ▼                                                                   │
│  ┌────────────────────────┐                                              │
│  │ COMPLETED (Error: err) │                                              │
│  └────────────────────────┘                                              │
│                                                                          │
│  Note: sync.Once ensures only one completion (Complete OR Fail)          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Handler Registration Timing

```
┌───────────────────────────────────────────────────────────────────────────┐
│  OnSuccess/OnFailure behavior depends on completion state:               │
│                                                                          │
│  Case 1: Handler registered BEFORE completion                            │
│  ─────────────────────────────────────────────                           │
│  f.OnSuccess(handler)  →  handler added to f.successHandlers             │
│  ...later...                                                             │
│  f.Complete(value)     →  handler(value) called                          │
│                                                                          │
│  Case 2: Handler registered AFTER completion                             │
│  ─────────────────────────────────────────────                           │
│  f.Complete(value)     →  f.completed = true                             │
│  ...later...                                                             │
│  f.OnSuccess(handler)  →  handler(value) called IMMEDIATELY              │
└───────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Confusing Spots → Suggested Fixes

### Issue #1: Dual Future Systems (Untyped + Typed)

**Location**: `reactive.go`, `async.go`

**Problem**:
```go
// Two parallel systems that don't interoperate well:
Future        // interface{} based - reactive.go
FutureT[T]    // generic based - async.go

// FutureT wraps Future internally:
type FutureT[T any] struct {
    future Future  // The untyped Future
}
```

Confusing which to use and how they relate.

**Suggested Fix**:
```go
// Document clearly:
// - Use FutureT[T] for new code (type-safe, Go-idiomatic)
// - Use Future only for compatibility with Vert.x patterns
// - FutureT internally wraps Future, so they're interoperable

// Or deprecate Future in favor of FutureT:
// Deprecated: Use FutureT[T] instead for type safety.
type Future interface { ... }
```

---

### Issue #2: `TryComplete` / `TryFail` Always Return True

**Location**: `reactive.go:267-274`

**Problem**:
```go
func (p *promise) TryComplete(result interface{}) bool {
    p.Complete(result)
    return true  // Always true! Should return false if already completed
}

func (p *promise) TryFail(err error) bool {
    p.Fail(err)
    return true  // Always true!
}
```

The "Try" prefix implies it might fail, but it always returns true.

**Suggested Fix**:
```go
func (p *promise) TryComplete(result interface{}) bool {
    // Track if this call actually completed the promise
    completed := false
    p.Future.(*future).once.Do(func() {
        completed = true
        // ... complete logic
    })
    return completed
}

// Or rename to clarify behavior:
func (p *promise) ForceComplete(result interface{}) {
    p.Complete(result)
}
```

---

### Issue #3: `Await` Can Miss Result Channel

**Location**: `reactive.go:188-211`

**Problem**:
```go
func (f *future) Await(ctx context.Context) (interface{}, error) {
    // Check if already completed
    f.mu.RLock()
    if f.completed {
        result := f.result
        f.mu.RUnlock()
        return result.Value, result.Error
    }
    f.mu.RUnlock()

    // RACE CONDITION: Future may complete between RUnlock and select
    select {
    case result := <-f.resultChan:  // resultChan is buffered(1), may already be consumed
        // ...
    }
}
```

If `Await` is called twice, second call may block forever.

**Suggested Fix**:
```go
func (f *future) Await(ctx context.Context) (interface{}, error) {
    // Always check completed state first (handles multiple Await calls)
    f.mu.RLock()
    if f.completed {
        result := f.result
        f.mu.RUnlock()
        if result.Error != nil {
            return nil, result.Error
        }
        return result.Value, nil
    }
    f.mu.RUnlock()

    // Wait with periodic recheck (handles race condition)
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case result := <-f.resultChan:
            // Put it back for other Await calls
            select {
            case f.resultChan <- result:
            default:
            }
            if result.Error != nil {
                return nil, result.Error
            }
            return result.Value, nil
        case <-ticker.C:
            f.mu.RLock()
            if f.completed {
                result := f.result
                f.mu.RUnlock()
                if result.Error != nil {
                    return nil, result.Error
                }
                return result.Value, nil
            }
            f.mu.RUnlock()
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
}
```

---

### Issue #4: `configInjectedVerticle` Injects on Every Start/Stop

**Location**: `main_verticle.go:133-145`

**Problem**:
```go
func (v *configInjectedVerticle) Start(ctx core.FluxorContext) error {
    for k, val := range v.cfg {
        ctx.SetConfig(k, val)  // Modifies ctx every time
    }
    return v.inner.Start(ctx)
}
```

If verticle is restarted, config is re-injected (benign but wasteful).

**Suggested Fix**:
```go
// Document the behavior:
// configInjectedVerticle injects config on every Start/Stop call.
// This is intentional to ensure config is available even if FluxorContext
// is replaced between restarts.

// Or inject once:
type configInjectedVerticle struct {
    inner    core.Verticle
    cfg      map[string]any
    injected bool
}

func (v *configInjectedVerticle) Start(ctx core.FluxorContext) error {
    if !v.injected {
        for k, val := range v.cfg {
            ctx.SetConfig(k, val)
        }
        v.injected = true
    }
    return v.inner.Start(ctx)
}
```

---

### Issue #5: `All` Executes Sequentially

**Location**: `async.go:276-295`

**Problem**:
```go
func All[T any](ctx context.Context, futures ...interface{}) *FutureT[[]T] {
    // ...
    go func() {
        results := make([]T, 0, len(futures))
        for _, f := range futures {
            result, err := f.Await(ctx)  // Sequential! Waits for each
            // ...
        }
    }()
}
```

Unlike JavaScript's `Promise.all()`, this executes sequentially.

**Suggested Fix**:
```go
func All[T any](ctx context.Context, futures ...interface{}) *FutureT[[]T] {
    promise := NewPromiseT[[]T]()

    go func() {
        results := make([]T, len(futures))
        errors := make([]error, len(futures))
        var wg sync.WaitGroup
        
        // Wait for all concurrently
        for i, f := range futures {
            wg.Add(1)
            go func(idx int, future interface{}) {
                defer wg.Done()
                result, err := future.(interface {
                    Await(context.Context) (T, error)
                }).Await(ctx)
                if err != nil {
                    errors[idx] = err
                } else {
                    results[idx] = result
                }
            }(i, f)
        }
        
        wg.Wait()
        
        // Check for any errors
        for _, err := range errors {
            if err != nil {
                promise.Fail(err)
                return
            }
        }
        promise.Complete(results)
    }()

    return &promise.FutureT
}
```

---

### Issue #6: `Race` Doesn't Cancel Losing Futures

**Location**: `async.go:297-338`

**Problem**:
```go
func Race[T any](ctx context.Context, futures ...interface{}) *FutureT[T] {
    // ...
    for _, f := range futures {
        go func(future ...) {
            result, err := future.Await(ctx)  // These continue running!
            // ...
        }(f)
    }
}
```

After first completes, other goroutines keep running.

**Suggested Fix**:
```go
func Race[T any](ctx context.Context, futures ...interface{}) *FutureT[T] {
    promise := NewPromiseT[T]()
    
    // Create cancellable context for race
    raceCtx, cancel := context.WithCancel(ctx)
    
    go func() {
        defer cancel()  // Cancel all other Awaits when done
        
        resultChan := make(chan T, 1)
        errChan := make(chan error, 1)

        for _, f := range futures {
            go func(future ...) {
                result, err := future.Await(raceCtx)  // Uses raceCtx!
                // ...
            }(f)
        }

        select {
        case result := <-resultChan:
            promise.Complete(result)
        case err := <-errChan:
            promise.Fail(err)
        case <-ctx.Done():
            promise.Fail(ctx.Err())
        }
    }()

    return &promise.FutureT
}
```

---

### Issue #7: ReactiveVerticle.ExecuteReactive Has Synchronous Request

**Location**: `reactive.go:291-313`

**Problem**:
```go
func (rv *ReactiveVerticle) ExecuteReactive(ctx context.Context, address string, data interface{}) Future {
    promise := NewPromise()

    // This is synchronous! Blocks until request completes
    msg, err := rv.vertx.EventBus().Request(address, data, 5*time.Second)
    if err != nil {
        promise.Fail(err)
        return promise
    }

    // Then wraps in goroutine (unnecessary)
    go func() {
        if msg.Body() != nil {
            promise.Complete(msg.Body())
        }
    }()
}
```

Defeats the purpose of reactive pattern.

**Suggested Fix**:
```go
func (rv *ReactiveVerticle) ExecuteReactive(ctx context.Context, address string, data interface{}) Future {
    promise := NewPromise()

    // Make the request async
    go func() {
        msg, err := rv.vertx.EventBus().Request(address, data, 5*time.Second)
        if err != nil {
            promise.Fail(err)
            return
        }
        if msg.Body() != nil {
            promise.Complete(msg.Body())
        } else {
            promise.Fail(&Error{Message: "no reply received"})
        }
    }()

    return promise
}
```

---

## Summary: Quick Reference

| Issue | Location | Severity | Fix Type |
|-------|----------|----------|----------|
| #1 Dual Future systems | reactive.go, async.go | Medium | Document/Deprecate |
| #2 TryComplete always true | reactive.go:267-274 | Medium | Fix return value |
| #3 Await race condition | reactive.go:188-211 | **High** | Fix race |
| #4 Config re-injection | main_verticle.go | Low | Document |
| #5 All is sequential | async.go:276 | **High** | Parallelize |
| #6 Race doesn't cancel | async.go:297 | Medium | Add cancellation |
| #7 ExecuteReactive sync | reactive.go:291 | Medium | Make async |

---

## Recommended Priority

1. **High Priority**: Issue #3 (Await race), Issue #5 (All sequential)
2. **Medium Priority**: Issues #1, #2, #6, #7
3. **Low Priority**: Issue #4 (documentation)
