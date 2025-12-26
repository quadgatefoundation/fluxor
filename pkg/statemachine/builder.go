package statemachine

import (
	"fmt"
	"time"
)

// Builder provides a fluent API for building state machine definitions.
type Builder struct {
	definition *StateMachineDefinition
	errors     []error
}

// NewBuilder creates a new state machine builder.
func NewBuilder(id, name string) *Builder {
	return &Builder{
		definition: &StateMachineDefinition{
			ID:          id,
			Name:        name,
			States:      make(map[StateType]*State),
			Transitions: make([]*Transition, 0),
			Metadata:    make(map[string]interface{}),
			Version:     "1.0",
		},
		errors: make([]error, 0),
	}
}

// WithDescription sets the state machine description.
func (b *Builder) WithDescription(description string) *Builder {
	b.definition.Description = description
	return b
}

// WithVersion sets the state machine version.
func (b *Builder) WithVersion(version string) *Builder {
	b.definition.Version = version
	return b
}

// WithMetadata adds metadata to the state machine.
func (b *Builder) WithMetadata(key string, value interface{}) *Builder {
	b.definition.Metadata[key] = value
	return b
}

// WithInitialState sets the initial state.
func (b *Builder) WithInitialState(state StateType) *Builder {
	b.definition.InitialState = state
	return b
}

// AddState adds a state to the state machine.
func (b *Builder) AddState(state *State) *Builder {
	if state.ID == "" {
		b.errors = append(b.errors, fmt.Errorf("state ID is required"))
		return b
	}
	b.definition.States[state.ID] = state
	return b
}

// AddStates adds multiple states to the state machine.
func (b *Builder) AddStates(states ...*State) *Builder {
	for _, state := range states {
		b.AddState(state)
	}
	return b
}

// AddTransition adds a transition to the state machine.
func (b *Builder) AddTransition(transition *Transition) *Builder {
	if transition.ID == "" {
		b.errors = append(b.errors, fmt.Errorf("transition ID is required"))
		return b
	}
	b.definition.Transitions = append(b.definition.Transitions, transition)
	return b
}

// AddTransitions adds multiple transitions to the state machine.
func (b *Builder) AddTransitions(transitions ...*Transition) *Builder {
	for _, transition := range transitions {
		b.AddTransition(transition)
	}
	return b
}

// Build builds the state machine definition.
func (b *Builder) Build() (*StateMachineDefinition, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder has %d errors: %v", len(b.errors), b.errors[0])
	}

	if err := validateDefinition(b.definition); err != nil {
		return nil, err
	}

	return b.definition, nil
}

// StateBuilder provides a fluent API for building states.
type StateBuilder struct {
	state *State
}

// NewState creates a new state builder.
func NewState(id StateType, name string) *StateBuilder {
	return &StateBuilder{
		state: &State{
			ID:       id,
			Name:     name,
			Metadata: make(map[string]interface{}),
		},
	}
}

// WithDescription sets the state description.
func (sb *StateBuilder) WithDescription(description string) *StateBuilder {
	sb.state.Description = description
	return sb
}

// AsFinal marks the state as final.
func (sb *StateBuilder) AsFinal() *StateBuilder {
	sb.state.IsFinal = true
	return sb
}

// OnEnter sets the on-enter action.
func (sb *StateBuilder) OnEnter(action StateAction) *StateBuilder {
	sb.state.OnEnter = action
	return sb
}

// OnExit sets the on-exit action.
func (sb *StateBuilder) OnExit(action StateAction) *StateBuilder {
	sb.state.OnExit = action
	return sb
}

// WithMetadata adds metadata to the state.
func (sb *StateBuilder) WithMetadata(key string, value interface{}) *StateBuilder {
	sb.state.Metadata[key] = value
	return sb
}

// Build builds the state.
func (sb *StateBuilder) Build() *State {
	return sb.state
}

// TransitionBuilder provides a fluent API for building transitions.
type TransitionBuilder struct {
	transition *Transition
}

// NewTransition creates a new transition builder.
func NewTransition(id string, from StateType, to StateType, event TransitionEvent) *TransitionBuilder {
	return &TransitionBuilder{
		transition: &Transition{
			ID:       id,
			From:     from,
			To:       to,
			Event:    event,
			Priority: 0,
			Metadata: make(map[string]interface{}),
		},
	}
}

// WithGuard sets the transition guard.
func (tb *TransitionBuilder) WithGuard(guard TransitionGuard) *TransitionBuilder {
	tb.transition.Guard = guard
	return tb
}

// WithAction sets the transition action.
func (tb *TransitionBuilder) WithAction(action TransitionAction) *TransitionBuilder {
	tb.transition.Action = action
	return tb
}

// WithPriority sets the transition priority.
func (tb *TransitionBuilder) WithPriority(priority int) *TransitionBuilder {
	tb.transition.Priority = priority
	return tb
}

// WithMetadata adds metadata to the transition.
func (tb *TransitionBuilder) WithMetadata(key string, value interface{}) *TransitionBuilder {
	tb.transition.Metadata[key] = value
	return tb
}

// Build builds the transition.
func (tb *TransitionBuilder) Build() *Transition {
	return tb.transition
}

// EventBuilder provides a fluent API for building events.
type EventBuilder struct {
	event *Event
}

// NewEvent creates a new event builder.
func NewEvent(name TransitionEvent) *EventBuilder {
	return &EventBuilder{
		event: &Event{
			Name:      name,
			Data:      make(map[string]interface{}),
			Timestamp: time.Now(),
		},
	}
}

// WithData adds data to the event.
func (eb *EventBuilder) WithData(key string, value interface{}) *EventBuilder {
	eb.event.Data[key] = value
	return eb
}

// WithDataMap sets the event data map.
func (eb *EventBuilder) WithDataMap(data map[string]interface{}) *EventBuilder {
	eb.event.Data = data
	return eb
}

// WithSource sets the event source.
func (eb *EventBuilder) WithSource(source string) *EventBuilder {
	eb.event.Source = source
	return eb
}

// WithRequestID sets the request ID.
func (eb *EventBuilder) WithRequestID(requestID string) *EventBuilder {
	eb.event.RequestID = requestID
	return eb
}

// WithTimestamp sets the event timestamp.
func (eb *EventBuilder) WithTimestamp(timestamp time.Time) *EventBuilder {
	eb.event.Timestamp = timestamp
	return eb
}

// Build builds the event.
func (eb *EventBuilder) Build() *Event {
	return eb.event
}

// Helper functions for common patterns

// SimpleState creates a simple state without actions.
func SimpleState(id StateType, name string) *State {
	return NewState(id, name).Build()
}

// FinalState creates a final state.
func FinalState(id StateType, name string) *State {
	return NewState(id, name).AsFinal().Build()
}

// SimpleTransition creates a simple transition without guards or actions.
func SimpleTransition(id string, from StateType, to StateType, event TransitionEvent) *Transition {
	return NewTransition(id, from, to, event).Build()
}

// ConditionalTransition creates a transition with a guard.
func ConditionalTransition(id string, from StateType, to StateType, event TransitionEvent, guard TransitionGuard) *Transition {
	return NewTransition(id, from, to, event).WithGuard(guard).Build()
}
