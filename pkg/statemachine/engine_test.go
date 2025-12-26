package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestEngine_CreateInstance(t *testing.T) {
	// Create a simple state machine
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, map[string]interface{}{
		"test": "data",
	})

	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	if instance.ID == "" {
		t.Error("Instance ID is empty")
	}

	if instance.Context.CurrentState != "initial" {
		t.Errorf("Expected initial state, got %s", instance.Context.CurrentState)
	}

	if instance.Status != InstanceStatusActive {
		t.Errorf("Expected active status, got %s", instance.Status)
	}
}

func TestEngine_SendEvent_SimpleTransition(t *testing.T) {
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Send event to transition
	event := NewEvent("start").Build()
	result, err := engine.SendEvent(ctx, instance.ID, event)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	if !result.Success {
		t.Error("Expected transition to succeed")
	}

	if result.FromState != "initial" {
		t.Errorf("Expected from state 'initial', got %s", result.FromState)
	}

	if result.ToState != "running" {
		t.Errorf("Expected to state 'running', got %s", result.ToState)
	}
}

func TestEngine_SendEvent_WithGuard(t *testing.T) {
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, map[string]interface{}{
		"can_complete": false,
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Transition to running
	event := NewEvent("start").Build()
	_, err = engine.SendEvent(ctx, instance.ID, event)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Try to complete (should fail due to guard)
	completeEvent := NewEvent("complete").Build()
	result, err := engine.SendEvent(ctx, instance.ID, completeEvent)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	if result.Success {
		t.Error("Expected transition to fail due to guard")
	}

	// Enable completion
	instance.Context.Data["can_complete"] = true

	// Try to complete again (should succeed)
	result, err = engine.SendEvent(ctx, instance.ID, completeEvent)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	if !result.Success {
		t.Error("Expected transition to succeed")
	}

	if result.ToState != "completed" {
		t.Errorf("Expected to state 'completed', got %s", result.ToState)
	}
}

func TestEngine_SendEvent_WithAction(t *testing.T) {
	actionCalled := false
	onEnterCalled := false
	onExitCalled := false

	builder := NewBuilder("test-actions", "Test Actions")
	builder.WithInitialState("initial")

	builder.AddStates(
		NewState("initial", "Initial").
			OnExit(func(ctx *StateContext) error {
				onExitCalled = true
				return nil
			}).
			Build(),
		NewState("next", "Next").
			OnEnter(func(ctx *StateContext) error {
				onEnterCalled = true
				return nil
			}).
			Build(),
	)

	builder.AddTransition(
		NewTransition("move", "initial", "next", "go").
			WithAction(func(ctx *StateContext, event *Event) error {
				actionCalled = true
				ctx.Data["action_data"] = "executed"
				return nil
			}).
			Build(),
	)

	def, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build state machine: %v", err)
	}

	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	event := NewEvent("go").Build()
	result, err := engine.SendEvent(ctx, instance.ID, event)
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	if !result.Success {
		t.Error("Expected transition to succeed")
	}

	if !actionCalled {
		t.Error("Expected transition action to be called")
	}

	if !onExitCalled {
		t.Error("Expected OnExit to be called")
	}

	if !onEnterCalled {
		t.Error("Expected OnEnter to be called")
	}

	if instance.Context.Data["action_data"] != "executed" {
		t.Error("Expected action data to be set")
	}
}

func TestEngine_FinalState(t *testing.T) {
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, map[string]interface{}{
		"can_complete": true,
	})
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Transition to running
	engine.SendEvent(ctx, instance.ID, NewEvent("start").Build())

	// Transition to completed (final state)
	result, err := engine.SendEvent(ctx, instance.ID, NewEvent("complete").Build())
	if err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	if !result.Success {
		t.Error("Expected transition to succeed")
	}

	if instance.Status != InstanceStatusCompleted {
		t.Errorf("Expected instance status to be completed, got %s", instance.Status)
	}

	if instance.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestEngine_History(t *testing.T) {
	def := createTestStateMachine()
	config := DefaultConfig()
	config.EnableHistory = true
	config.MaxHistorySize = 10

	engine, err := NewEngine(def, config, nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Make a transition
	engine.SendEvent(ctx, instance.ID, NewEvent("start").Build())

	if len(instance.Context.History) == 0 {
		t.Error("Expected history to be recorded")
	}

	entry := instance.Context.History[0]
	if entry.FromState != "initial" {
		t.Errorf("Expected from state 'initial', got %s", entry.FromState)
	}

	if entry.ToState != "running" {
		t.Errorf("Expected to state 'running', got %s", entry.ToState)
	}

	if entry.Event != "start" {
		t.Errorf("Expected event 'start', got %s", entry.Event)
	}
}

func TestEngine_CanTransition(t *testing.T) {
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Check if can transition with valid event
	event := NewEvent("start").Build()
	can, err := engine.CanTransition(ctx, instance.ID, event)
	if err != nil {
		t.Fatalf("Failed to check transition: %v", err)
	}

	if !can {
		t.Error("Expected to be able to transition")
	}

	// Check if can transition with invalid event
	invalidEvent := NewEvent("invalid").Build()
	can, err = engine.CanTransition(ctx, instance.ID, invalidEvent)
	if err != nil {
		t.Fatalf("Failed to check transition: %v", err)
	}

	if can {
		t.Error("Expected to not be able to transition")
	}
}

func TestEngine_GetCurrentState(t *testing.T) {
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	state, err := engine.GetCurrentState(ctx, instance.ID)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	if state != "initial" {
		t.Errorf("Expected initial state, got %s", state)
	}

	// Transition
	engine.SendEvent(ctx, instance.ID, NewEvent("start").Build())

	state, err = engine.GetCurrentState(ctx, instance.ID)
	if err != nil {
		t.Fatalf("Failed to get current state: %v", err)
	}

	if state != "running" {
		t.Errorf("Expected running state, got %s", state)
	}
}

func TestBuilder_Validation(t *testing.T) {
	// Test missing initial state
	builder := NewBuilder("test", "Test")
	builder.AddState(SimpleState("state1", "State 1"))
	builder.AddTransition(SimpleTransition("t1", "state1", "state2", "event1"))

	_, err := builder.Build()
	if err == nil {
		t.Error("Expected error for missing initial state")
	}

	// Test invalid transition (references unknown state)
	builder2 := NewBuilder("test2", "Test 2")
	builder2.WithInitialState("state1")
	builder2.AddState(SimpleState("state1", "State 1"))
	builder2.AddTransition(SimpleTransition("t1", "state1", "unknown", "event1"))

	_, err = builder2.Build()
	if err == nil {
		t.Error("Expected error for transition referencing unknown state")
	}
}

// Helper function to create a test state machine
func createTestStateMachine() *StateMachineDefinition {
	builder := NewBuilder("test-sm", "Test State Machine")
	builder.WithInitialState("initial")

	builder.AddStates(
		SimpleState("initial", "Initial State"),
		SimpleState("running", "Running State"),
		FinalState("completed", "Completed State"),
	)

	builder.AddTransitions(
		SimpleTransition("start", "initial", "running", "start"),
		NewTransition("complete", "running", "completed", "complete").
			WithGuard(func(ctx *StateContext, event *Event) (bool, error) {
				canComplete, ok := ctx.Data["can_complete"].(bool)
				return ok && canComplete, nil
			}).
			Build(),
	)

	def, err := builder.Build()
	if err != nil {
		panic(err)
	}

	return def
}

func TestEngine_EventBusIntegration(t *testing.T) {
	// Create a test Vertx and EventBus
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	def := createTestStateMachine()
	config := DefaultConfig()
	config.EnableEventBus = true

	engine, err := NewEngine(def, config, vertx.EventBus())
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()
	instance, err := engine.CreateInstance(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	// Send event via EventBus
	req := map[string]interface{}{
		"instanceId": instance.ID,
		"event":      "start",
		"data":       map[string]interface{}{},
	}

	// Give EventBus time to register consumers
	time.Sleep(100 * time.Millisecond)

	msg, err := vertx.EventBus().Request("statemachine.test-sm.event", req, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to send event via EventBus: %v", err)
	}

	var resp map[string]interface{}
	body, ok := msg.Body().([]byte)
	if !ok {
		t.Fatalf("Invalid message body type")
	}

	if err := core.JSONDecode(body, &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if success, ok := resp["success"].(bool); !ok || !success {
		t.Error("Expected event to succeed")
	}
}

func BenchmarkEngine_CreateInstance(b *testing.B) {
	def := createTestStateMachine()
	engine, _ := NewEngine(def, DefaultConfig(), nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CreateInstance(ctx, nil)
	}
}

func BenchmarkEngine_SendEvent(b *testing.B) {
	def := createTestStateMachine()
	engine, _ := NewEngine(def, DefaultConfig(), nil)
	ctx := context.Background()

	// Create instances
	instances := make([]*StateMachineInstance, b.N)
	for i := 0; i < b.N; i++ {
		instances[i], _ = engine.CreateInstance(ctx, nil)
	}

	event := NewEvent("start").Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.SendEvent(ctx, instances[i].ID, event)
	}
}
