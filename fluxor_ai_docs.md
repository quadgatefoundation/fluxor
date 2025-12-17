# Fluxor Runtime: AI-Oriented Technical Documentation

## 1. Introduction

This document provides a technical overview of the `fluxor` runtime, designed for another AI to understand its architecture, core principles, and intended usage patterns. The `fluxor` runtime is a Go-based framework for building robust, scalable, and concurrent applications using a message-passing, actor-like model.

## 2. Core Concepts & Principles

The runtime is built on several key principles that ensure predictable and resilient behavior.

### 2.1. Actor Model & Components

- **Core Unit:** The fundamental building block in `fluxor` is the `Component` (`types.Component`).
- **Isolation:** Each `Component` is designed to be an isolated, independent unit of execution, similar to an actor in the Actor Model. It manages its own state and communicates with other components exclusively through asynchronous messages.
- **No Shared State:** Components MUST NOT share state directly. All inter-component communication is brokered by the Event Bus.

### 2.2. Reactor: The Heartbeat of a Component

- **Single-Threaded Execution:** Each `Component` is driven by its own `Reactor` (`pkg/reactor`). The `Reactor` provides a single-threaded, serialized execution environment. This completely eliminates the need for locks or other concurrency primitives within a `Component`'s business logic, dramatically simplifying development.
- **Mailbox:** Each `Reactor` has a `mailbox`, which is a buffered channel that queues incoming functions to be executed serially.
- **Non-Blocking Execution:** The `Reactor.Execute` method is **non-blocking**. It immediately attempts to queue the function. If the mailbox is full, it returns `types.ErrBackpressure` without waiting.

### 2.3. Backpressure Management

Backpressure is a core, non-negotiable principle. No part of the system is allowed to block indefinitely. When a part of the system is overloaded, it signals this overload to the caller, allowing the system to shed load gracefully instead of crashing.

- **`types.ErrBackpressure`:** This error is the primary mechanism for signaling overload. It is returned by `Reactor.Execute` and `WorkerPool.Submit` when their respective queues are full.

### 2.4. Worker Pool: Offloading Blocking Work

- **Purpose:** The `WorkerPool` (`pkg/worker`) manages a fixed-size pool of goroutines for executing long-running, blocking, or CPU-intensive tasks.
- **Decoupling:** It decouples `Reactors` from blocking operations. A `Component` should never perform a blocking operation directly on its `Reactor` thread. Instead, it should submit the task to the `WorkerPool`.
- **Non-Blocking Submission:** Like the `Reactor`, `WorkerPool.Submit` is **non-blocking** and returns `types.ErrBackpressure` if its job queue is at capacity.

### 2.5. Event Bus: The Central Nervous System

- **Asynchronous Communication:** The `localBus` (`pkg/bus`) facilitates all communication between components.
- **Reactor-Correct Dispatch:** When `Send` is called, the bus does not execute the message handler directly. It identifies the recipient `Component`'s `Reactor` and uses `Reactor.Execute` to schedule the message delivery. This ensures the message is processed on the recipient's dedicated thread, respecting the single-threaded execution guarantee.
- **Subscription:** Components subscribe to topics using `Subscribe(topic, componentName, mailbox)`. The `componentName` is crucial for the bus to look up the correct `Reactor` via the `ReactorProvider`.

## 3. Architecture & Data Flow

### 3.1. The `Runtime` Orchestrator

The `Runtime` (`pkg/runtime`) is the top-level container that initializes, manages, and orchestrates all other parts of the system.

- **Initialization:** On `NewRuntime`, it creates the `WorkerPool`, the `localBus`, and the `reactorStore` (`bus.ReactorProvider`). It wires them together by setting the provider on the bus.
- **Deployment (`Deploy`):** When a `Component` is deployed:
    1. A new `Reactor` is created specifically for that component.
    2. The `Component` and its `Reactor` are registered in the `Runtime`'s stores.
- **Lifecycle (`Start`/`Stop`):** The `Runtime` manages the lifecycle of all registered components and core services. `Start` iterates through all components, starting their associated `Reactors` and then the components themselves. `Stop` performs the teardown in the reverse order.

### 3.2. Message Flow: A `Send` Operation

1.  **Component A (`Sender`)** calls `bus.Send("topic.B", message)` from within its `Reactor` thread.
2.  The `localBus` looks up subscribers for `"topic.B"`.
3.  It selects **Component B (`Receiver`)** as the recipient.
4.  The bus retrieves Component B's `componentName` from its subscription data.
5.  It queries the `ReactorProvider` (`reactorStore`) with `GetReactor("ComponentBName")` to get Component B's `Reactor`.
6.  The bus calls `ComponentB_Reactor.Execute(function)` where `function` contains the logic to push the `message` into Component B's mailbox channel.
7.  The call to `Execute` returns immediately (either `nil` or `ErrBackpressure`) to the bus, and subsequently to Component A. Component A's `Reactor` is never blocked.
8.  At some point in the future, Component B's `Reactor` loop picks up the function from its mailbox and executes it, delivering the message to the component's handler.

## 4. Code Structure

- **`/pkg/types`**: Defines the core interfaces (`Component`, `Bus`) and shared data structures (`Message`, `Mailbox`) that decouple all other packages.
- **`/pkg/reactor`**: Contains the `Reactor` implementation, providing the single-threaded execution context for components.
- **`/pkg/worker`**: Contains the `WorkerPool` for offloading blocking tasks.
- **`/pkg/bus`**: Contains the `localBus` implementation for message passing and the `ReactorProvider` for connecting the bus to the correct reactors.
- **`/pkg/runtime`**: Contains the `Runtime` which integrates all the above components into a cohesive application framework.
