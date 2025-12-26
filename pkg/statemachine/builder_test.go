package statemachine

import (
	"testing"
	"time"
)

func TestBuilder_Basic(t *testing.T) {
	builder := NewBuilder("test-builder", "Test Builder")
	builder.WithDescription("Test description").
		WithVersion("1.0").
		WithInitialState("start")

	builder.AddState(SimpleState("start", "Start State"))
	builder.AddState(FinalState("end", "End State"))
	builder.AddTransition(SimpleTransition("t1", "start", "end", "finish"))

	def, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	if def.ID != "test-builder" {
		t.Errorf("Expected ID 'test-builder', got %s", def.ID)
	}

	if def.Name != "Test Builder" {
		t.Errorf("Expected name 'Test Builder', got %s", def.Name)
	}

	if def.Description != "Test description" {
		t.Errorf("Expected description, got %s", def.Description)
	}

	if def.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", def.Version)
	}

	if len(def.States) != 2 {
		t.Errorf("Expected 2 states, got %d", len(def.States))
	}

	if len(def.Transitions) != 1 {
		t.Errorf("Expected 1 transition, got %d", len(def.Transitions))
	}
}

func TestStateBuilder(t *testing.T) {
	enterCalled := false
	exitCalled := false

	state := NewState("test", "Test State").
		WithDescription("A test state").
		AsFinal().
		OnEnter(func(ctx *StateContext) error {
			enterCalled = true
			return nil
		}).
		OnExit(func(ctx *StateContext) error {
			exitCalled = true
			return nil
		}).
		WithMetadata("key", "value").
		Build()

	if state.ID != "test" {
		t.Errorf("Expected ID 'test', got %s", state.ID)
	}

	if state.Name != "Test State" {
		t.Errorf("Expected name 'Test State', got %s", state.Name)
	}

	if state.Description != "A test state" {
		t.Errorf("Expected description, got %s", state.Description)
	}

	if !state.IsFinal {
		t.Error("Expected state to be final")
	}

	if state.OnEnter == nil {
		t.Error("Expected OnEnter to be set")
	}

	if state.OnExit == nil {
		t.Error("Expected OnExit to be set")
	}

	// Test actions
	ctx := &StateContext{Data: make(map[string]interface{})}
	state.OnEnter(ctx)
	if !enterCalled {
		t.Error("Expected OnEnter to be called")
	}

	state.OnExit(ctx)
	if !exitCalled {
		t.Error("Expected OnExit to be called")
	}

	if state.Metadata["key"] != "value" {
		t.Error("Expected metadata to be set")
	}
}

func TestTransitionBuilder(t *testing.T) {
	guardCalled := false
	actionCalled := false

	transition := NewTransition("test", "from", "to", "event").
		WithGuard(func(ctx *StateContext, event *Event) (bool, error) {
			guardCalled = true
			return true, nil
		}).
		WithAction(func(ctx *StateContext, event *Event) error {
			actionCalled = true
			return nil
		}).
		WithPriority(10).
		WithMetadata("key", "value").
		Build()

	if transition.ID != "test" {
		t.Errorf("Expected ID 'test', got %s", transition.ID)
	}

	if transition.From != "from" {
		t.Errorf("Expected from 'from', got %s", transition.From)
	}

	if transition.To != "to" {
		t.Errorf("Expected to 'to', got %s", transition.To)
	}

	if transition.Event != "event" {
		t.Errorf("Expected event 'event', got %s", transition.Event)
	}

	if transition.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", transition.Priority)
	}

	if transition.Guard == nil {
		t.Error("Expected guard to be set")
	}

	if transition.Action == nil {
		t.Error("Expected action to be set")
	}

	// Test guard and action
	ctx := &StateContext{Data: make(map[string]interface{})}
	event := &Event{Data: make(map[string]interface{})}

	result, _ := transition.Guard(ctx, event)
	if !guardCalled {
		t.Error("Expected guard to be called")
	}
	if !result {
		t.Error("Expected guard to return true")
	}

	transition.Action(ctx, event)
	if !actionCalled {
		t.Error("Expected action to be called")
	}

	if transition.Metadata["key"] != "value" {
		t.Error("Expected metadata to be set")
	}
}

func TestEventBuilder(t *testing.T) {
	now := time.Now()

	event := NewEvent("test-event").
		WithData("key1", "value1").
		WithData("key2", 123).
		WithSource("test-source").
		WithRequestID("req-123").
		WithTimestamp(now).
		Build()

	if event.Name != "test-event" {
		t.Errorf("Expected name 'test-event', got %s", event.Name)
	}

	if event.Data["key1"] != "value1" {
		t.Error("Expected key1 to be set")
	}

	if event.Data["key2"] != 123 {
		t.Error("Expected key2 to be set")
	}

	if event.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", event.Source)
	}

	if event.RequestID != "req-123" {
		t.Errorf("Expected request ID 'req-123', got %s", event.RequestID)
	}

	if !event.Timestamp.Equal(now) {
		t.Error("Expected timestamp to match")
	}
}

func TestEventBuilder_WithDataMap(t *testing.T) {
	dataMap := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	event := NewEvent("test").
		WithDataMap(dataMap).
		Build()

	if len(event.Data) != 3 {
		t.Errorf("Expected 3 data entries, got %d", len(event.Data))
	}

	if event.Data["key1"] != "value1" {
		t.Error("Expected key1 to be set")
	}

	if event.Data["key2"] != 123 {
		t.Error("Expected key2 to be set")
	}

	if event.Data["key3"] != true {
		t.Error("Expected key3 to be set")
	}
}

func TestSimpleState(t *testing.T) {
	state := SimpleState("test", "Test")

	if state.ID != "test" {
		t.Errorf("Expected ID 'test', got %s", state.ID)
	}

	if state.Name != "Test" {
		t.Errorf("Expected name 'Test', got %s", state.Name)
	}

	if state.IsFinal {
		t.Error("Expected state not to be final")
	}
}

func TestFinalState(t *testing.T) {
	state := FinalState("done", "Done")

	if state.ID != "done" {
		t.Errorf("Expected ID 'done', got %s", state.ID)
	}

	if state.Name != "Done" {
		t.Errorf("Expected name 'Done', got %s", state.Name)
	}

	if !state.IsFinal {
		t.Error("Expected state to be final")
	}
}

func TestConditionalTransition(t *testing.T) {
	guardCalled := false
	guard := func(ctx *StateContext, event *Event) (bool, error) {
		guardCalled = true
		return true, nil
	}

	transition := ConditionalTransition("test", "from", "to", "event", guard)

	if transition.Guard == nil {
		t.Error("Expected guard to be set")
	}

	ctx := &StateContext{Data: make(map[string]interface{})}
	event := &Event{Data: make(map[string]interface{})}

	transition.Guard(ctx, event)
	if !guardCalled {
		t.Error("Expected guard to be called")
	}
}

func TestBuilder_AddMultiple(t *testing.T) {
	builder := NewBuilder("test", "Test")
	builder.WithInitialState("s1")

	states := []*State{
		SimpleState("s1", "State 1"),
		SimpleState("s2", "State 2"),
		SimpleState("s3", "State 3"),
	}

	transitions := []*Transition{
		SimpleTransition("t1", "s1", "s2", "e1"),
		SimpleTransition("t2", "s2", "s3", "e2"),
	}

	builder.AddStates(states...)
	builder.AddTransitions(transitions...)

	def, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	if len(def.States) != 3 {
		t.Errorf("Expected 3 states, got %d", len(def.States))
	}

	if len(def.Transitions) != 2 {
		t.Errorf("Expected 2 transitions, got %d", len(def.Transitions))
	}
}

func TestBuilder_WithMetadata(t *testing.T) {
	builder := NewBuilder("test", "Test")
	builder.WithMetadata("author", "John Doe").
		WithMetadata("version", "1.0").
		WithInitialState("start")

	builder.AddState(SimpleState("start", "Start"))

	def, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build: %v", err)
	}

	if def.Metadata["author"] != "John Doe" {
		t.Error("Expected author metadata")
	}

	if def.Metadata["version"] != "1.0" {
		t.Error("Expected version metadata")
	}
}
