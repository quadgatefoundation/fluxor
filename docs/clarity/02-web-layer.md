# Web Layer Deep Dive: FastHTTPServer, Router, Backpressure

This document provides a detailed analysis of `pkg/web/` focusing on:
1. Request flow from accept to response
2. Backpressure mechanism and CCU-based configuration
3. Router and middleware chain
4. **Confusing spots → Suggested fixes**

---

## 1. Request Flow

### High-Level Flow

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                    FastHTTPServer                        │
                    └─────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  handleRequest(ctx *fasthttp.RequestCtx)                                    │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Step 1: BackpressureController.TryAcquire()                           │  │
│  │         → If full: return 503 immediately (fail-fast)                 │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                              │ success                                       │
│                              ▼                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Step 2: requestMailbox.Send(ctx)                                      │  │
│  │         → If full: release backpressure, return 503 (fail-fast)       │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                              │ success                                       │
│                              ▼                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Request queued → queuedRequests++                                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  Worker Pool (processRequestFromMailbox)                                    │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ 1. requestMailbox.Receive(ctx) - blocking                             │  │
│  │ 2. queuedRequests--                                                   │  │
│  │ 3. processRequest(ctx) with panic isolation                           │  │
│  │ 4. backpressure.Release() - always, even on panic                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  processRequest(ctx *fasthttp.RequestCtx)                                   │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ 1. Generate/extract X-Request-ID                                      │  │
│  │ 2. Create FastRequestContext (wraps fasthttp.RequestCtx)              │  │
│  │ 3. router.ServeFastHTTP(reqCtx)                                       │  │
│  │ 4. Track metrics (totalRequests, successfulRequests, errorRequests)   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  FastRouter.ServeFastHTTP(ctx *FastRequestContext)                          │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ 1. Match route (method + path pattern)                                │  │
│  │ 2. Extract path params (:id → params["id"])                           │  │
│  │ 3. Build middleware chain (route-specific + global)                   │  │
│  │ 4. Execute handler                                                    │  │
│  │ 5. On error: return 500                                               │  │
│  │ 6. On no match: return 404                                            │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Backpressure Mechanism

### Two-Stage Backpressure

The server uses **two-stage backpressure** for protection:

```
Stage 1: BackpressureController (CCU-based)
├── Tracks current concurrent connections
├── normalCapacity = target utilization (e.g., 67% of max)
├── If currentLoad >= normalCapacity → reject immediately (503)
└── TryAcquire() / Release() pattern

Stage 2: Request Mailbox (Queue-based)
├── Bounded queue (MaxQueue from config)
├── If queue full → reject immediately (503)
└── Non-blocking send (mailbox.Send())
```

### CCU Configuration Formulas

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  CCUBasedConfig(addr, maxCCU, overflowCCU)                                  │
│  ─────────────────────────────────────────                                  │
│  workers    = maxCCU / 10          (clamped: 50-500)                        │
│  queueSize  = maxCCU - workers     (min: 100)                               │
│  maxConns   = maxCCU + overflowCCU                                          │
│  normalCap  = queueSize + workers  (= maxCCU)                               │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  CCUBasedConfigWithUtilization(addr, maxCCU, utilizationPercent)            │
│  ─────────────────────────────────────────────────────────────              │
│  normalCap  = maxCCU * (utilizationPercent / 100)                           │
│  workers    = normalCap / 10       (clamped: 50-500)                        │
│  queueSize  = normalCap - workers  (min: 100)                               │
│  maxConns   = maxCCU               (allows 100% connections)                │
│                                                                             │
│  Example: maxCCU=10000, utilization=67%                                     │
│  → normalCap=6700, workers=500, queue=6200                                  │
│  → Under normal load: 67% utilization                                       │
│  → Under spike: can accept up to 10000 conns, but 503 after 6700            │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Metrics

| Metric | Description |
|--------|-------------|
| `QueuedRequests` | Currently queued requests |
| `RejectedRequests` | Total 503 responses (backpressure) |
| `QueueUtilization` | queued / maxQueue * 100% |
| `NormalCCU` | Target capacity (e.g., 6700) |
| `CurrentCCU` | Current load from backpressure controller |
| `CCUUtilization` | currentLoad / normalCapacity * 100% |
| `TotalRequests` | All requests (success + error + rejected) |
| `SuccessfulRequests` | 2xx responses |
| `ErrorRequests` | 5xx responses |

---

## 3. Router Architecture

### Two Router Implementations

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  router (pkg/web/router.go)           │  FastRouter (pkg/web/fast_router.go)│
│  ─────────────────────────────────────│─────────────────────────────────────│
│  - Standard net/http                  │  - fasthttp-based                   │
│  - http.Handler interface             │  - FastRequestHandler               │
│  - RequestContext                     │  - FastRequestContext               │
│  - Middleware applied at Route()      │  - Middleware applied at Serve()    │
│  - Slower, more compatible            │  - Faster, less memory              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Middleware Chain Order

```
Registration:
  r.UseFast(mw1)
  r.UseFast(mw2)
  r.GETFast("/path", handler)

Execution order (onion model):
  mw1 → mw2 → handler → mw2 → mw1

For route-specific middleware:
  r.GETFastWith("/path", handler, routeMw1, routeMw2)

Execution order:
  globalMw1 → globalMw2 → routeMw1 → routeMw2 → handler
```

---

## 4. Confusing Spots → Suggested Fixes

### Issue #1: Dual Router Interface Confusion

**Location**: `router.go`, `fast_router.go`

**Problem**:
```go
// FastRouter has both interfaces but only FastXXX methods work
func (r *FastRouter) GET(path string, handler RequestHandler) {
    // Not implemented for standard http - use GETFast instead
}
```

Empty implementations are confusing.

**Suggested Fix**:
```go
// Option A: Remove empty methods, don't implement Router interface
// Option B: Panic with clear message
func (r *FastRouter) GET(path string, handler RequestHandler) {
    panic("FastRouter.GET not implemented - use GETFast instead")
}

// Option C: Actually implement conversion
func (r *FastRouter) GET(path string, handler RequestHandler) {
    r.GETFast(path, convertHandler(handler))
}
```

---

### Issue #2: Middleware Application Timing Differs

**Location**: `router.go:52-66`, `fast_router.go:54-61`

**Problem**:
- `router.Route()`: Middleware applied at registration time (stored with handler)
- `FastRouter.ServeFastHTTP()`: Middleware applied at request time

This causes subtle behavior differences.

**Suggested Fix**:
```go
// Document the difference clearly:
// router.go
// Route registers a handler with middleware applied immediately.
// Middleware added after Route() won't affect this handler.
func (r *router) Route(method, path string, handler RequestHandler) {
    // ...
}

// fast_router.go  
// ServeFastHTTP applies middleware at request time.
// All middleware (global + route) is applied on every request.
func (r *FastRouter) ServeFastHTTP(ctx *FastRequestContext) {
    // ...
}
```

---

### Issue #3: `normalCapacity` vs `MaxQueue` + `Workers`

**Location**: `fasthttp_server.go:153`, `backpressure.go:12`

**Problem**:
```go
// In NewFastHTTPServer:
normalCapacity := config.MaxQueue + config.Workers

// But backpressure tracks CCU, not queue+workers
// This conflates two concepts:
// 1. Queue capacity (how many requests can wait)
// 2. CCU capacity (how many concurrent connections)
```

**Suggested Fix**:
```go
// Rename for clarity:
type FastHTTPServerConfig struct {
    Addr            string
    QueueCapacity   int // How many requests can wait in queue
    WorkerCount     int // Number of worker goroutines
    MaxCCU          int // Max concurrent connections (backpressure limit)
    // ...
}

// Or document the relationship:
// normalCapacity = QueueCapacity + WorkerCount = effective CCU limit
```

---

### Issue #4: Double Panic Recovery

**Location**: `fasthttp_server.go:344-349`, `fasthttp_server.go:374-392`

**Problem**:
```go
func (s *FastHTTPServer) processRequestFromMailbox(ctx context.Context) error {
    defer func() {
        if r := recover(); r != nil {  // First recovery
            s.Logger().Errorf("panic in worker (isolated): %v", r)
        }
    }()
    
    for {
        // ...
        func() {
            defer func() {
                if r := recover(); r != nil {  // Second recovery (per-request)
                    // ...
                }
            }()
            s.processRequest(reqCtx)
        }()
    }
}
```

Two levels of panic recovery - outer loop and inner request.

**Suggested Fix**:
```go
// Document the intentional double recovery:
// processRequestFromMailbox has two panic recovery layers:
// 1. Outer: Protects the worker loop itself (shouldn't panic, but defense-in-depth)
// 2. Inner: Isolates individual request panics (expected, per-request isolation)
//
// The inner recovery writes a 500 response; outer just logs.
```

---

### Issue #5: `FastRequestContext.Context()` Creates New Context

**Location**: `fasthttp_server.go:536-542`

**Problem**:
```go
func (c *FastRequestContext) Context() context.Context {
    ctx := context.Background()  // Creates NEW context every call!
    if c.requestID != "" {
        ctx = core.WithRequestID(ctx, c.requestID)
    }
    return ctx
}
```

Creates a new context every call, which:
- Doesn't chain with Vertx's root context
- Loses deadline/cancellation from server shutdown

**Suggested Fix**:
```go
type FastRequestContext struct {
    // ...
    goCtx context.Context // Cached context, created once
}

// In processRequest:
reqCtx := &FastRequestContext{
    // ...
    goCtx: core.WithRequestID(s.Vertx().Context(), requestID),
}

func (c *FastRequestContext) Context() context.Context {
    return c.goCtx // Return cached context
}
```

---

### Issue #6: `ServerMetrics` Has Redundant Fields

**Location**: `fasthttp_server.go:264-276`

**Problem**:
```go
type ServerMetrics struct {
    QueuedRequests   int64   // Current queued
    QueueCapacity    int     // Max queue
    QueueUtilization float64 // queued/capacity * 100 - redundant!
    
    NormalCCU        int     // From backpressure
    CurrentCCU       int     // From backpressure  
    CCUUtilization   float64 // current/normal * 100 - redundant!
}
```

Utilization can be calculated from other fields.

**Suggested Fix**:
```go
// Option A: Remove redundant fields, let consumers calculate
type ServerMetrics struct {
    QueuedRequests   int64
    QueueCapacity    int
    NormalCCU        int
    CurrentCCU       int
    // Utilization calculated by consumer
}

// Option B: Keep for convenience but document
type ServerMetrics struct {
    QueuedRequests   int64
    QueueCapacity    int
    QueueUtilization float64 // Convenience: QueuedRequests / QueueCapacity * 100
    // ...
}
```

---

### Issue #7: `BackpressureController` Reset Logic

**Location**: `backpressure.go:36-41`

**Problem**:
```go
func (bc *BackpressureController) TryAcquire() bool {
    now := time.Now().Unix()
    if now-bc.lastReset > bc.resetInterval {
        atomic.StoreInt64(&bc.currentLoad, 0)  // DANGER: Resets load to 0!
        atomic.StoreInt64(&bc.lastReset, now)
    }
    // ...
}
```

Periodically resetting `currentLoad` to 0 can cause:
- Active requests "disappear" from tracking
- Backpressure becomes ineffective after reset
- If Release() is called after reset, load goes negative

**Suggested Fix**:
```go
// Option A: Remove the reset (track actual concurrent requests)
// The current acquire/release pattern should balance itself

// Option B: Reset only the rejectedCount (metrics), not currentLoad
func (bc *BackpressureController) TryAcquire() bool {
    now := time.Now().Unix()
    if now-bc.lastReset > bc.resetInterval {
        // Only reset metrics, not currentLoad
        atomic.StoreInt64(&bc.rejectedCount, 0)
        atomic.StoreInt64(&bc.lastReset, now)
    }
    // ...
}
```

---

## Summary: Quick Reference

| Issue | Location | Severity | Fix Type |
|-------|----------|----------|----------|
| #1 Empty Router methods | fast_router.go | Medium | Panic/Remove |
| #2 Middleware timing differs | router.go, fast_router.go | Low | Document |
| #3 normalCapacity naming | fasthttp_server.go | Medium | Rename/Document |
| #4 Double panic recovery | fasthttp_server.go | Low | Document |
| #5 Context() creates new ctx | fasthttp_server.go:536 | High | Cache context |
| #6 Redundant metrics fields | fasthttp_server.go | Low | Remove/Document |
| #7 Backpressure reset bug | backpressure.go:36 | **High** | Fix reset logic |

---

## Recommended Priority

1. **High Priority**: Issue #7 (backpressure reset bug), Issue #5 (context creation)
2. **Medium Priority**: Issues #1, #3 (API clarity)
3. **Low Priority**: Issues #2, #4, #6 (documentation)
