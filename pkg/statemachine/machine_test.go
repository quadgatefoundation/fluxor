package statemachine

import (
	"context"
	"testing"
	"time"
)

func TestStateMachine_BasicTransitions(t *testing.T) {
	// Build a simple state machine: idle -> running -> completed
	builder := NewBuilder("test-machine").
		Name("Test Machine").
		InitialState("idle")
	
	builder.State("idle").
		On("start", "running").Done().
		Done()
	
	builder.State("running").
		On("complete", "completed").Done().
		On("cancel", "cancelled").Done().
		Done()
	
	builder.State("completed").
		Final(true).
		Done()
	
	builder.State("cancelled").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	if err := sm.Start(ctx); err != nil {
		t.Fatalf("Failed to start state machine: %v", err)
	}

	// Check initial state
	if sm.CurrentState() != "idle" {
		t.Errorf("Expected initial state 'idle', got '%s'", sm.CurrentState())
	}

	// Transition to running
	err = sm.Send(ctx, Event{Name: "start", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to send 'start' event: %v", err)
	}

	if sm.CurrentState() != "running" {
		t.Errorf("Expected state 'running', got '%s'", sm.CurrentState())
	}

	// Transition to completed
	err = sm.Send(ctx, Event{Name: "complete", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to send 'complete' event: %v", err)
	}

	if sm.CurrentState() != "completed" {
		t.Errorf("Expected state 'completed', got '%s'", sm.CurrentState())
	}

	// Check history
	history := sm.GetHistory()
	if len(history) != 2 {
		t.Errorf("Expected 2 history entries, got %d", len(history))
	}
}

func TestStateMachine_Guards(t *testing.T) {
	guardAllowed := true

	builder := NewBuilder("guard-test").
		InitialState("start")
	
	builder.State("start").
		On("proceed", "end").
			Guard(func(ctx context.Context, event Event) (bool, error) {
				return guardAllowed, nil
			}).
			Done().
		Done()
	
	builder.State("end").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	// Guard rejects
	guardAllowed = false
	err = sm.Send(ctx, Event{Name: "proceed", Timestamp: time.Now()})
	if err == nil {
		t.Error("Expected guard to reject transition")
	}

	if sm.CurrentState() != "start" {
		t.Errorf("State should not have changed, got '%s'", sm.CurrentState())
	}

	// Guard allows
	guardAllowed = true
	err = sm.Send(ctx, Event{Name: "proceed", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Guard should have allowed transition: %v", err)
	}

	if sm.CurrentState() != "end" {
		t.Errorf("Expected state 'end', got '%s'", sm.CurrentState())
	}
}

func TestStateMachine_Actions(t *testing.T) {
	actionExecuted := false

	builder := NewBuilder("action-test").
		InitialState("start")
	
	builder.State("start").
		On("proceed", "end").
			Action(func(ctx context.Context, from string, to string, event Event) error {
				actionExecuted = true
				return nil
			}).
			Done().
		Done()
	
	builder.State("end").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	err = sm.Send(ctx, Event{Name: "proceed", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to send event: %v", err)
	}

	if !actionExecuted {
		t.Error("Action was not executed")
	}
}

func TestStateMachine_EntryExitHandlers(t *testing.T) {
	entryExecuted := false
	exitExecuted := false

	builder := NewBuilder("handler-test").
		InitialState("start")
	
	builder.State("start").
		Exit(func(ctx context.Context, event Event) error {
			exitExecuted = true
			return nil
		}).
		On("proceed", "end").Done().
		Done()
	
	builder.State("end").
		Entry(func(ctx context.Context, event Event) error {
			entryExecuted = true
			return nil
		}).
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	err = sm.Send(ctx, Event{Name: "proceed", Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to send event: %v", err)
	}

	if !exitExecuted {
		t.Error("Exit handler was not executed")
	}

	if !entryExecuted {
		t.Error("Entry handler was not executed")
	}
}

func TestStateMachine_InvalidTransition(t *testing.T) {
	builder := NewBuilder("invalid-test").
		InitialState("start")
	
	builder.State("start").
		On("valid", "end").Done().
		Done()
	
	builder.State("end").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	// Try invalid event
	err = sm.Send(ctx, Event{Name: "invalid", Timestamp: time.Now()})
	if err == nil {
		t.Error("Expected error for invalid transition")
	}

	if smErr, ok := err.(*StateMachineError); ok {
		if smErr.Code != ErrorCodeInvalidTransition {
			t.Errorf("Expected ErrorCodeInvalidTransition, got %v", smErr.Code)
		}
	} else {
		t.Error("Expected StateMachineError")
	}
}

func TestStateMachine_Reset(t *testing.T) {
	builder := NewBuilder("reset-test").
		InitialState("idle")
	
	builder.State("idle").
		On("start", "running").Done().
		Done()
	
	builder.State("running").
		On("complete", "done").Done().
		Done()
	
	builder.State("done").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	// Progress through states
	sm.Send(ctx, Event{Name: "start", Timestamp: time.Now()})
	sm.Send(ctx, Event{Name: "complete", Timestamp: time.Now()})

	if sm.CurrentState() != "done" {
		t.Errorf("Expected state 'done', got '%s'", sm.CurrentState())
	}

	// Reset
	err = sm.Reset(ctx)
	if err != nil {
		t.Errorf("Failed to reset: %v", err)
	}

	if sm.CurrentState() != "idle" {
		t.Errorf("Expected state 'idle' after reset, got '%s'", sm.CurrentState())
	}

	if len(sm.GetHistory()) != 0 {
		t.Error("History should be cleared after reset")
	}
}

func TestStateMachine_CanTransition(t *testing.T) {
	builder := NewBuilder("can-transition-test").
		InitialState("idle")
	
	builder.State("idle").
		On("start", "running").Done().
		Done()
	
	builder.State("running").
		On("stop", "idle").Done().
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	if !sm.CanTransition("start") {
		t.Error("Should be able to transition on 'start' event")
	}

	if sm.CanTransition("stop") {
		t.Error("Should not be able to transition on 'stop' event from idle")
	}
}

func TestStateMachine_AsyncSend(t *testing.T) {
	builder := NewBuilder("async-test").
		InitialState("idle")
	
	builder.State("idle").
		On("start", "running").Done().
		Done()
	
	builder.State("running").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build definition: %v", err)
	}

	sm, err := NewStateMachine(def)
	if err != nil {
		t.Fatalf("Failed to create state machine: %v", err)
	}

	ctx := context.Background()
	sm.Start(ctx)

	future := sm.SendAsync(ctx, Event{Name: "start", Timestamp: time.Now()})

	err = future.Await(ctx)
	if err != nil {
		t.Errorf("Async send failed: %v", err)
	}

	if sm.CurrentState() != "running" {
		t.Errorf("Expected state 'running', got '%s'", sm.CurrentState())
	}
}

func TestBuilder_ComplexMachine(t *testing.T) {
	builder := NewBuilder("order-machine").
		Name("Order Processing Machine").
		Description("Handles order lifecycle").
		InitialState("pending")
	
	builder.State("pending").
		On("approve", "approved").
			Priority(10).
			Guard(DataFieldExists("orderId")).
			Action(LogAction(func(msg string) { t.Log(msg) })).
			Done().
		On("reject", "rejected").Done().
		Done()
	
	builder.State("approved").
		On("ship", "shipped").Done().
		On("cancel", "cancelled").Done().
		Done()
	
	builder.State("shipped").
		Final(true).
		Done()
	
	builder.State("rejected").
		Final(true).
		Done()
	
	builder.State("cancelled").
		Final(true).
		Done()
	
	def, err := builder.Build()

	if err != nil {
		t.Fatalf("Failed to build complex machine: %v", err)
	}

	if def.InitialState != "pending" {
		t.Errorf("Expected initial state 'pending', got '%s'", def.InitialState)
	}

	if len(def.States) != 5 {
		t.Errorf("Expected 5 states, got %d", len(def.States))
	}
}
