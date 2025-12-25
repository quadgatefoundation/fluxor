# Terminology (Glossary)

This repo mixes framework code, examples, and docs. To avoid confusion, use the following **standard terms** consistently in code comments and documentation.

## Core runtime

- **Fluxor**: the overall framework/runtime in this repository.
- **Vertx**: the Fluxor runtime instance (`core.Vertx`). Use **`Vertx`** (not `VertX`, not `Vert.x`).
- **Verticle**: a deployable unit/component (`core.Verticle`). Use **`Verticle`**.
- **EventBus**: the messaging abstraction (`core.EventBus`). Use **`EventBus`** (not `Event Bus` / `event bus`).
- **Message**: an EventBus message (`core.Message`).

## HTTP layer

- **FastHTTPServer**: the fasthttp-based server (`web.FastHTTPServer`). Use **`FastHTTPServer`**.
- **FastRouter**: the route/middleware layer (`web.FastRouter`). Use **`FastRouter`**.
- **FastRequestContext**: request context wrapper (`web.FastRequestContext`). Use **`FastRequestContext`**.

## IDs, tracing, metrics

- **Request ID**: spelled as **“Request ID”**; header name is **`X-Request-ID`**; code symbol is usually `requestID`.
- **OpenTelemetry**: spelled **OpenTelemetry** (OTel is acceptable as abbreviation).
- **Prometheus**: spelled **Prometheus**.

## Examples vs framework

- **Framework code**: `pkg/` (and `cmd/` for binaries).
- **Examples**: under `examples/` (e.g. `examples/fluxor-project`, `examples/todo-api`).

