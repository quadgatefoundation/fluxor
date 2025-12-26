package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestStateMachineBasicFlow(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine
	engine := NewEngine(vertx.EventBus())

	// Build simple state machine
	machine := NewStateMachineBuilder("test", "Test Machine").
		InitialState("start").
		AddState("start", "Start State").
			AddTransition("next", "middle").Done().
			Done().
		AddState("middle", "Middle State").
			AddTransition("finish", "end").Done().
			Done().
		AddState("end", "End State").
			Final(true).
			Done().
		Build()

	// Register machine
	if err := engine.RegisterMachine(machine); err != nil {
		t.Fatalf("failed to register machine: %v", err)
	}

	// Create instance
	ctx := context.Background()
	instanceID, err := engine.CreateInstance(ctx, "test", map[string]interface{}{
		"data": "test",
	})
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	// Verify initial state
	instance, err := engine.GetInstance(instanceID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}
	if instance.CurrentState != "start" {
		t.Errorf("expected start state, got %s", instance.CurrentState)
	}

	// Send event
	event := &Event{
		Name:      "next",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}
	if err := engine.SendEvent(ctx, instanceID, event); err != nil {
		t.Fatalf("failed to send event: %v", err)
	}

	// Verify transition
	instance, _ = engine.GetInstance(instanceID)
	if instance.CurrentState != "middle" {
		t.Errorf("expected middle state, got %s", instance.CurrentState)
	}

	// Send final event
	event = &Event{
		Name:      "finish",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}
	if err := engine.SendEvent(ctx, instanceID, event); err != nil {
		t.Fatalf("failed to send final event: %v", err)
	}

	// Verify final state
	instance, _ = engine.GetInstance(instanceID)
	if instance.CurrentState != "end" {
		t.Errorf("expected end state, got %s", instance.CurrentState)
	}
	if instance.Status != ExecutionStatusCompleted {
		t.Errorf("expected completed status, got %s", instance.Status)
	}

	// Verify history
	if len(instance.History) != 2 {
		t.Errorf("expected 2 transitions in history, got %d", len(instance.History))
	}
}

func TestStateMachineWithGuards(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine
	engine := NewEngine(vertx.EventBus())

	// Register guard
	engine.RegisterGuard("positiveAmount", func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		if event.Data == nil {
			return false
		}
		amount, ok := event.Data["amount"].(float64)
		return ok && amount > 0
	})

	// Build state machine with guard
	machine := NewStateMachineBuilder("guarded", "Guarded Machine").
		InitialState("start").
		AddState("start", "Start").
			AddTransition("process", "processing").
				Guard("positiveAmount").
				Done().
			Done().
		AddState("processing", "Processing").
			Final(true).
			Done().
		Build()

	engine.RegisterMachine(machine)

	ctx := context.Background()
	instanceID, _ := engine.CreateInstance(ctx, "guarded", nil)

	// Send event with negative amount (should fail guard)
	event := &Event{
		Name:      "process",
		Data:      map[string]interface{}{"amount": -10.0},
		Timestamp: time.Now(),
	}
	engine.SendEvent(ctx, instanceID, event)

	instance, _ := engine.GetInstance(instanceID)
	if instance.CurrentState != "start" {
		t.Errorf("expected to remain in start state, got %s", instance.CurrentState)
	}

	// Send event with positive amount (should pass guard)
	event = &Event{
		Name:      "process",
		Data:      map[string]interface{}{"amount": 10.0},
		Timestamp: time.Now(),
	}
	engine.SendEvent(ctx, instanceID, event)

	instance, _ = engine.GetInstance(instanceID)
	if instance.CurrentState != "processing" {
		t.Errorf("expected processing state, got %s", instance.CurrentState)
	}
}

func TestStateMachineWithActions(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine
	engine := NewEngine(vertx.EventBus())

	actionCalled := false

	// Register action
	engine.RegisterAction("testAction", func(ctx context.Context, event *Event, execCtx *ExecutionContext) error {
		actionCalled = true
		execCtx.Variables["processed"] = true
		return nil
	})

	// Build state machine with action
	machine := NewStateMachineBuilder("action-test", "Action Test").
		InitialState("start").
		AddState("start", "Start").
			AddTransition("process", "processing").
				Action("testAction").
				Done().
			Done().
		AddState("processing", "Processing").
			Final(true).
			Done().
		Build()

	engine.RegisterMachine(machine)

	ctx := context.Background()
	instanceID, _ := engine.CreateInstance(ctx, "action-test", nil)

	// Send event
	event := &Event{
		Name:      "process",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}
	engine.SendEvent(ctx, instanceID, event)

	// Verify action was called
	if !actionCalled {
		t.Error("action was not called")
	}

	// Verify variables were set
	instance, _ := engine.GetInstance(instanceID)
	if processed, ok := instance.Variables["processed"].(bool); !ok || !processed {
		t.Error("action did not set variable")
	}
}

func TestStateMachineTransitionPriority(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine
	engine := NewEngine(vertx.EventBus())

	// Register guards
	engine.RegisterGuard("alwaysTrue", func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		return true
	})
	engine.RegisterGuard("alwaysFalse", func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		return false
	})

	// Build state machine with prioritized transitions
	machine := NewStateMachineBuilder("priority-test", "Priority Test").
		InitialState("start").
		AddState("start", "Start").
			// Lower priority, always true - should not be taken
			AddTransition("go", "low").
				Guard("alwaysTrue").
				Priority(1).
				Done().
			// Higher priority, always false - should be evaluated first but fail
			AddTransition("go", "high").
				Guard("alwaysFalse").
				Priority(10).
				Done().
			Done().
		AddState("high", "High Priority").Final(true).Done().
		AddState("low", "Low Priority").Final(true).Done().
		Build()

	engine.RegisterMachine(machine)

	ctx := context.Background()
	instanceID, _ := engine.CreateInstance(ctx, "priority-test", nil)

	// Send event
	event := &Event{
		Name:      "go",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}
	engine.SendEvent(ctx, instanceID, event)

	// Should take lower priority transition since higher priority guard failed
	instance, _ := engine.GetInstance(instanceID)
	if instance.CurrentState != "low" {
		t.Errorf("expected low state (lower priority with passing guard), got %s", instance.CurrentState)
	}
}

func TestStateMachinePersistence(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine with memory persistence
	engine := NewEngine(vertx.EventBus())
	persistence := NewMemoryPersistenceProvider()
	engine.SetPersistence(persistence)

	// Build simple machine
	machine := NewStateMachineBuilder("persist", "Persist Test").
		InitialState("start").
		AddState("start", "Start").
			AddTransition("next", "end").Done().
			Done().
		AddState("end", "End").Final(true).Done().
		Build()

	engine.RegisterMachine(machine)

	// Create instance
	ctx := context.Background()
	instanceID, _ := engine.CreateInstance(ctx, "persist", map[string]interface{}{
		"data": "test",
	})

	// Save instance
	if err := engine.SaveInstance(instanceID); err != nil {
		t.Fatalf("failed to save instance: %v", err)
	}

	// Delete from engine
	delete(engine.instances, instanceID)

	// Restore instance
	if err := engine.RestoreInstance(instanceID); err != nil {
		t.Fatalf("failed to restore instance: %v", err)
	}

	// Verify restored
	instance, err := engine.GetInstance(instanceID)
	if err != nil {
		t.Fatalf("failed to get restored instance: %v", err)
	}
	if instance.CurrentState != "start" {
		t.Errorf("expected start state after restore, got %s", instance.CurrentState)
	}
	if data, ok := instance.Data["data"].(string); !ok || data != "test" {
		t.Error("data not restored correctly")
	}
}

func TestStateChangeListener(t *testing.T) {
	// Create EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Create engine
	engine := NewEngine(vertx.EventBus())

	changeCount := 0
	var lastFrom, lastTo string

	// Add listener
	engine.AddStateChangeListener(func(ctx context.Context, instanceID, from, to string, event *Event) {
		changeCount++
		lastFrom = from
		lastTo = to
	})

	// Build machine
	machine := NewStateMachineBuilder("listener", "Listener Test").
		InitialState("start").
		AddState("start", "Start").
			AddTransition("next", "end").Done().
			Done().
		AddState("end", "End").Final(true).Done().
		Build()

	engine.RegisterMachine(machine)

	ctx := context.Background()
	instanceID, _ := engine.CreateInstance(ctx, "listener", nil)

	// Wait for initial state notification
	time.Sleep(50 * time.Millisecond)

	// Send event
	event := &Event{
		Name:      "next",
		Data:      map[string]interface{}{},
		Timestamp: time.Now(),
	}
	engine.SendEvent(ctx, instanceID, event)

	// Wait for listener
	time.Sleep(50 * time.Millisecond)

	// Verify listener was called
	if changeCount < 2 { // Initial state + transition
		t.Errorf("expected at least 2 state changes, got %d", changeCount)
	}
	if lastFrom != "start" || lastTo != "end" {
		t.Errorf("expected transition from start to end, got %s -> %s", lastFrom, lastTo)
	}
}
