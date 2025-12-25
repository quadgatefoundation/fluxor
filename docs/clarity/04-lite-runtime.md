# Lite Runtime Deep Dive: Lightweight Fluxor Implementation

This document provides a detailed analysis of `pkg/lite/` focusing on:
1. Architecture comparison: Full vs Lite
2. Component overview: Bus, WorkerPool, App
3. Web layer: Router, HttpVerticle, FastRouter
4. **Confusing spots → Suggested fixes**

---

## 1. Architecture Comparison

### Full Fluxor vs Lite

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Full Fluxor (pkg/core)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Vertx with deployment state machine                                      │
│  • EventBus with Request/Reply, validation, JSON encoding                   │
│  • FluxorContext with config injection                                      │
│  • Executor/Mailbox abstractions                                            │
│  • Comprehensive error handling                                             │
│  • Request ID propagation                                                   │
│  • ~3000 LOC in core                                                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                            Lite Fluxor (pkg/lite)                           │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Simple App with component list                                           │
│  • Fire-and-forget Bus (no reply, no validation)                            │
│  • FluxorContext with Bus + WorkerPool                                      │
│  • Simple WorkerPool (raw goroutines)                                       │
│  • Minimal error handling                                                   │
│  • No request ID tracking                                                   │
│  • ~500 LOC total                                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

### When to Use Which

| Use Case | Recommendation |
|----------|----------------|
| Production microservices | Full Fluxor |
| Prototyping / learning | Lite |
| High-throughput, simple logic | Lite + webfast |
| Distributed systems | Full Fluxor (NATS integration) |
| Single-process tools | Lite |

---

## 2. Core Components

### Bus (`pkg/lite/core/bus.go`)

**Purpose**: Simple in-process pub/sub

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Bus                                                                         │
│  ├── subs: map[string]map[uint64]func(any)  // topic → id → handler          │
│  ├── nextID: uint64 (atomic)                // subscription ID counter       │
│  └── mu: sync.RWMutex                       // concurrent access             │
└─────────────────────────────────────────────────────────────────────────────┘

Subscribe(topic, handler) → unsubscribe func()
  │
  ├── Get unique ID (atomic)
  ├── Lock, add to subs[topic][id]
  └── Return closure that removes subscription

Publish(topic, msg)
  │
  ├── RLock, copy handlers
  ├── RUnlock
  └── For each handler: go h(msg)  // Fire-and-forget goroutine
```

**Key Differences from EventBus**:
- No validation
- No JSON encoding
- No Request/Reply
- Fire-and-forget (spawns goroutine per message)
- No backpressure

---

### WorkerPool (`pkg/lite/core/worker.go`)

**Purpose**: Simple bounded goroutine pool

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  WorkerPool                                                                  │
│  └── tasks: chan func()   // buffered channel (queueSize)                   │
└─────────────────────────────────────────────────────────────────────────────┘

NewWorkerPool(workerCount, queueSize)
  │
  ├── Create buffered channel
  └── Spawn workerCount goroutines:
      for task := range tasks { task() }

Submit(task func())
  │
  └── tasks <- task  // Blocks if full (no backpressure feedback)

Shutdown()
  │
  └── close(tasks)  // Workers exit when channel drains
```

**Key Differences from Executor**:
- No context cancellation
- No error handling
- Blocking Submit (no TrySubmit)
- No stats/metrics

---

### App (`pkg/lite/fluxor/runtime.go`)

**Purpose**: Lightweight application runtime

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  App                                                                         │
│  ├── bus: *Bus                                                               │
│  ├── worker: *WorkerPool (10 workers, 1024 queue)                           │
│  ├── ctx: context.Context                                                   │
│  ├── cancel: context.CancelFunc                                             │
│  ├── deployments: []Component                                               │
│  └── mu: sync.Mutex                                                         │
└─────────────────────────────────────────────────────────────────────────────┘

New() → *App
  │
  ├── Create Bus
  ├── Create WorkerPool(10, 1024)
  └── Create cancellable context

Deploy(c Component)
  │
  ├── Generate UUID
  ├── Create FluxorContext
  ├── c.OnStart(ctx)
  └── Append to deployments

Run()
  │
  ├── Wait for SIGINT/SIGTERM
  ├── Cancel context
  ├── Call OnStop for all deployments
  └── Shutdown worker pool
```

**Key Differences from MainVerticle/Vertx**:
- No deployment state machine
- No config injection
- No async deploy support
- Fixed worker pool config
- Simple component list (not map)

---

## 3. Web Layer

### Standard HTTP (`pkg/lite/web/`)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Router (net/http based)                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  routes: []*route                                                           │
│  middleware: []Middleware                                                   │
│  notFound: HandlerFunc                                                      │
│  onError: func(c, err) error                                                │
└─────────────────────────────────────────────────────────────────────────────┘

Handle(c *fx.Context) error
  │
  ├── Match route by method + pattern
  ├── Extract path params
  ├── Build middleware chain (route then global)
  ├── Execute handler
  └── On error: call onError handler

HttpVerticle
  │
  ├── OnStart: Start http.Server in goroutine
  └── OnStop: Close server
```

### Fast HTTP (`pkg/lite/webfast/`)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Router (fasthttp based)                                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  getRoutes: []*route                                                        │
│  postRoutes: []*route                                                       │
│  middleware: []Middleware                                                   │
│  coreCtx: *FluxorContext                                                    │
│  paramPool: sync.Pool                                                       │
└─────────────────────────────────────────────────────────────────────────────┘

Optimizations:
  • Method dispatch via byte comparison (not string)
  • Pre-compiled route segments
  • Param slice pooling (sync.Pool)
  • Zero-copy param values (unsafe.String)
  • Separate route slices per method
```

---

## 4. Context Types

### Comparison

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  pkg/lite/core/FluxorContext                                                │
│  ├── id: string                                                             │
│  ├── bus: *Bus                                                              │
│  ├── worker: *WorkerPool                                                    │
│  ├── stdCtx: context.Context                                                │
│  └── logger: *slog.Logger                                                   │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  pkg/lite/fx/Context (for web)                                              │
│  ├── W: http.ResponseWriter                                                 │
│  ├── R: *http.Request                                                       │
│  ├── Params: map[string]string                                              │
│  └── coreCtx: *FluxorContext                                                │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  pkg/lite/fx/FastContext (for webfast)                                      │
│  ├── RC: *fasthttp.RequestCtx                                               │
│  ├── Params: []Param  (slice, not map - pooled)                             │
│  └── coreCtx: *FluxorContext                                                │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Confusing Spots → Suggested Fixes

### Issue #1: Bus.Publish Creates Unbounded Goroutines

**Location**: `pkg/lite/core/bus.go:57-59`

**Problem**:
```go
for _, h := range handlers {
    go h(msg)  // New goroutine per handler per message!
}
```

Under high load, this can create thousands of goroutines.

**Suggested Fix**:
```go
// Option A: Use the worker pool
for _, h := range handlers {
    h := h  // Capture for closure
    msg := msg
    b.worker.Submit(func() { h(msg) })
}

// Option B: Synchronous delivery (simplest, no goroutine explosion)
for _, h := range handlers {
    h(msg)  // Synchronous, but blocks publisher
}

// Option C: Document the limitation
// Bus is fire-and-forget with unbounded goroutines.
// For production use, consider pkg/core/EventBus instead.
```

---

### Issue #2: WorkerPool.Submit Blocks Without Feedback

**Location**: `pkg/lite/core/worker.go:28-30`

**Problem**:
```go
func (wp *WorkerPool) Submit(task func()) {
    wp.tasks <- task  // Blocks if channel full!
}
```

No way to know if submission blocked or would block.

**Suggested Fix**:
```go
// Option A: Add TrySubmit
func (wp *WorkerPool) TrySubmit(task func()) bool {
    select {
    case wp.tasks <- task:
        return true
    default:
        return false  // Queue full
    }
}

// Option B: Add timeout
func (wp *WorkerPool) SubmitWithTimeout(task func(), timeout time.Duration) bool {
    select {
    case wp.tasks <- task:
        return true
    case <-time.After(timeout):
        return false
    }
}
```

---

### Issue #3: No Context Cancellation in WorkerPool

**Location**: `pkg/lite/core/worker.go:17-23`

**Problem**:
```go
for i := 0; i < workerCount; i++ {
    go func() {
        for task := range wp.tasks {
            task()  // No way to interrupt a running task
        }
    }()
}
```

Workers don't respect context cancellation.

**Suggested Fix**:
```go
func NewWorkerPoolWithContext(ctx context.Context, workerCount, queueSize int) *WorkerPool {
    wp := &WorkerPool{tasks: make(chan func(), queueSize)}
    for i := 0; i < workerCount; i++ {
        go func() {
            for {
                select {
                case task, ok := <-wp.tasks:
                    if !ok {
                        return
                    }
                    task()
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
    return wp
}
```

---

### Issue #4: App.Deploy Doesn't Return Deployment ID

**Location**: `pkg/lite/fluxor/runtime.go:45-57`

**Problem**:
```go
func (a *App) Deploy(c core.Component) {
    id := uuid.New().String()
    // ...
    // ID is generated but not returned!
}
```

No way to identify or undeploy a specific component.

**Suggested Fix**:
```go
func (a *App) Deploy(c core.Component) (string, error) {
    id := uuid.New().String()
    fctx := core.NewFluxorContext(a.ctx, a.bus, a.worker, id)

    if err := c.OnStart(fctx); err != nil {
        return "", fmt.Errorf("deploy failed: %w", err)
    }

    a.mu.Lock()
    a.deployments = append(a.deployments, &deployment{id: id, component: c})
    a.mu.Unlock()

    return id, nil
}

func (a *App) Undeploy(id string) error {
    // Find and remove by ID
}
```

---

### Issue #5: App.Run Ignores OnStop Errors

**Location**: `pkg/lite/fluxor/runtime.go:73-75`

**Problem**:
```go
for _, c := range deps {
    _ = c.OnStop()  // Error silently ignored!
}
```

Errors during shutdown are discarded.

**Suggested Fix**:
```go
for _, c := range deps {
    if err := c.OnStop(); err != nil {
        fmt.Printf("⚠️ Stop error for component: %v\n", err)
        // Or collect errors and return at end
    }
}
```

---

### Issue #6: FastRouter.Bind() Must Be Called Before Use

**Location**: `pkg/lite/webfast/router.go:65-67`

**Problem**:
```go
func (r *Router) Bind(coreCtx *core.FluxorContext) {
    r.coreCtx = coreCtx
}
```

If `Bind()` is not called, `r.coreCtx` is nil and causes panic.

**Suggested Fix**:
```go
// Option A: Require in constructor
func NewRouter(coreCtx *core.FluxorContext) *Router {
    if coreCtx == nil {
        panic("coreCtx required")
    }
    r := &Router{coreCtx: coreCtx, ...}
    return r
}

// Option B: Fail-fast in Handler
func (r *Router) Handler() fasthttp.RequestHandler {
    if r.coreCtx == nil {
        panic("Router.Bind() must be called before Handler()")
    }
    return func(rc *fasthttp.RequestCtx) { ... }
}
```

---

### Issue #7: Zero-Copy Params Lifetime Issue

**Location**: `pkg/lite/webfast/router.go:218-219`, `pkg/lite/fx/fast_context.go:15-18`

**Problem**:
```go
// In router.go:
c.Params = append(c.Params, fx.Param{Key: seg.param, Value: b2s(part)})

// b2s uses unsafe pointer to request memory:
func b2s(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}
```

Param values point to request memory - invalid after request completes.

**Suggested Fix**:
```go
// Option A: Document clearly (already done, but emphasize)
// Param is a path parameter. WARNING: Value is backed by request memory
// in litefast. DO NOT store beyond request lifetime. Copy if needed:
//   stored := strings.Clone(param.Value)

// Option B: Add safe accessor
func (c *FastContext) ParamCopy(key string) string {
    v := c.Param(key)
    return strings.Clone(v)  // Makes a copy
}
```

---

### Issue #8: HttpVerticle.OnStop Doesn't Wait for Connections

**Location**: `pkg/lite/web/http_verticle.go:46-50`

**Problem**:
```go
func (v *HttpVerticle) OnStop() error {
    if v.server != nil {
        return v.server.Close()  // Immediate close, no graceful shutdown
    }
    return nil
}
```

Active connections are immediately terminated.

**Suggested Fix**:
```go
func (v *HttpVerticle) OnStop() error {
    if v.server == nil {
        return nil
    }
    
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := v.server.Shutdown(ctx); err != nil {
        // Fallback to immediate close
        return v.server.Close()
    }
    return nil
}
```

---

## Summary: Quick Reference

| Issue | Location | Severity | Fix Type |
|-------|----------|----------|----------|
| #1 Unbounded goroutines | bus.go:57 | **High** | Use worker pool |
| #2 Submit blocks silently | worker.go:28 | Medium | Add TrySubmit |
| #3 No context cancellation | worker.go:17 | Medium | Add context support |
| #4 Deploy doesn't return ID | runtime.go:45 | Medium | Return (string, error) |
| #5 OnStop errors ignored | runtime.go:73 | Low | Log errors |
| #6 Bind() not required | router.go:65 | Medium | Require in constructor |
| #7 Unsafe param lifetime | router.go:218 | Medium | Document/Copy accessor |
| #8 No graceful shutdown | http_verticle.go:46 | Medium | Use Shutdown() |

---

## Recommended Priority

1. **High Priority**: Issue #1 (goroutine explosion risk)
2. **Medium Priority**: Issues #2, #3, #4, #6, #8 (API improvements)
3. **Low Priority**: Issues #5, #7 (documentation/convenience)

---

## Design Philosophy

The lite package is intentionally minimal:

> **"Lite is for when you need a 100-line app, not a 1000-line framework"**

Trade-offs:
- **Simplicity over safety**: No validation, less error handling
- **Speed over features**: Fewer abstractions, direct channel use
- **Explicit over implicit**: No magic config injection
- **Fire-and-forget over guaranteed delivery**: Simpler mental model

Use lite for:
- Quick prototypes
- Simple tools/scripts
- Learning Fluxor concepts
- High-throughput, simple request handling

Graduate to full Fluxor for:
- Production services
- Distributed systems
- Complex business logic
- Observability requirements
