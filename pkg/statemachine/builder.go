package statemachine

import (
	"context"
	"fmt"
	"time"
)

// Builder provides a fluent API for building state machines.
type Builder struct {
	definition    *Definition
	currentState  *stateBuilder
	err           error
}

// stateBuilder builds a single state.
type stateBuilder struct {
	parent *Builder
	state  *State
}

// transitionBuilder builds a single transition.
type transitionBuilder struct {
	parent     *stateBuilder
	transition *Transition
}

// NewBuilder creates a new state machine builder.
func NewBuilder(id string) *Builder {
	return &Builder{
		definition: &Definition{
			ID:     id,
			States: make(map[string]*State),
		},
	}
}

// Name sets the state machine name.
func (b *Builder) Name(name string) *Builder {
	b.definition.Name = name
	return b
}

// Description sets the state machine description.
func (b *Builder) Description(desc string) *Builder {
	b.definition.Description = desc
	return b
}

// Version sets the state machine version.
func (b *Builder) Version(version string) *Builder {
	b.definition.Version = version
	return b
}

// InitialState sets the initial state.
func (b *Builder) InitialState(state string) *Builder {
	b.definition.InitialState = state
	return b
}

// Context sets the shared context data.
func (b *Builder) Context(ctx map[string]interface{}) *Builder {
	b.definition.Context = ctx
	return b
}

// State adds a new state to the machine.
func (b *Builder) State(name string) *stateBuilder {
	state := &State{
		Name:        name,
		Transitions: make([]*Transition, 0),
		Metadata:    make(map[string]interface{}),
	}
	
	sb := &stateBuilder{
		parent: b,
		state:  state,
	}
	
	b.currentState = sb
	return sb
}

// Build builds and returns the state machine definition.
func (b *Builder) Build() (*Definition, error) {
	// Add current state if not yet added
	if b.currentState != nil && b.currentState.state != nil {
		b.definition.States[b.currentState.state.Name] = b.currentState.state
		b.currentState = nil
	}

	if b.err != nil {
		return nil, b.err
	}

	// Validate
	if err := validateDefinition(b.definition); err != nil {
		return nil, fmt.Errorf("invalid state machine definition: %w", err)
	}

	return b.definition, nil
}

// BuildAndCreate builds the definition and creates a state machine instance.
func (b *Builder) BuildAndCreate(opts ...Option) (StateMachine, error) {
	def, err := b.Build()
	if err != nil {
		return nil, err
	}
	return NewStateMachine(def, opts...)
}

// =============== stateBuilder methods ===============

// Final marks this state as a final state.
func (sb *stateBuilder) Final(isFinal bool) *stateBuilder {
	sb.state.IsFinal = isFinal
	return sb
}

// Entry sets the entry handler for this state.
func (sb *stateBuilder) Entry(handler Handler) *stateBuilder {
	sb.state.Entry = handler
	return sb
}

// Exit sets the exit handler for this state.
func (sb *stateBuilder) Exit(handler Handler) *stateBuilder {
	sb.state.Exit = handler
	return sb
}

// Metadata sets metadata for this state.
func (sb *stateBuilder) Metadata(key string, value interface{}) *stateBuilder {
	sb.state.Metadata[key] = value
	return sb
}

// On adds a transition triggered by an event.
func (sb *stateBuilder) On(event string, to string) *transitionBuilder {
	transition := &Transition{
		Event:    event,
		From:     sb.state.Name,
		To:       to,
		Priority: 0,
		Metadata: make(map[string]interface{}),
	}
	
	sb.state.Transitions = append(sb.state.Transitions, transition)
	
	return &transitionBuilder{
		parent:     sb,
		transition: transition,
	}
}

// Done finishes building this state and returns to the main builder.
func (sb *stateBuilder) Done() *Builder {
	// Add state to definition
	sb.parent.definition.States[sb.state.Name] = sb.state
	sb.parent.currentState = nil
	return sb.parent
}

// =============== transitionBuilder methods ===============

// Guard sets a guard condition for the transition.
func (tb *transitionBuilder) Guard(guard Guard) *transitionBuilder {
	tb.transition.Guard = guard
	return tb
}

// Action sets an action for the transition.
func (tb *transitionBuilder) Action(action Action) *transitionBuilder {
	tb.transition.Action = action
	return tb
}

// Priority sets the priority for the transition.
func (tb *transitionBuilder) Priority(priority int) *transitionBuilder {
	tb.transition.Priority = priority
	return tb
}

// Timeout sets the timeout for the transition action.
func (tb *transitionBuilder) Timeout(timeout time.Duration) *transitionBuilder {
	tb.transition.Timeout = timeout
	return tb
}

// Metadata sets metadata for the transition.
func (tb *transitionBuilder) Metadata(key string, value interface{}) *transitionBuilder {
	tb.transition.Metadata[key] = value
	return tb
}

// Done finishes building this transition and returns to the state builder.
func (tb *transitionBuilder) Done() *stateBuilder {
	return tb.parent
}

// OnDone is a convenience method to finish the transition and add another transition.
func (tb *transitionBuilder) OnDone(event string, to string) *transitionBuilder {
	return tb.parent.On(event, to)
}

// =============== Common Guard Functions ===============

// AlwaysAllow is a guard that always allows the transition.
func AlwaysAllow() Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		return true, nil
	}
}

// NeverAllow is a guard that never allows the transition.
func NeverAllow() Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		return false, nil
	}
}

// DataFieldEquals creates a guard that checks if event data field equals a value.
func DataFieldEquals(field string, value interface{}) Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		if event.Data == nil {
			return false, nil
		}
		fieldValue, ok := event.Data[field]
		if !ok {
			return false, nil
		}
		return fieldValue == value, nil
	}
}

// DataFieldExists creates a guard that checks if event data field exists.
func DataFieldExists(field string) Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		if event.Data == nil {
			return false, nil
		}
		_, ok := event.Data[field]
		return ok, nil
	}
}

// AndGuard combines multiple guards with AND logic.
func AndGuard(guards ...Guard) Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		for _, guard := range guards {
			allowed, err := guard(ctx, event)
			if err != nil {
				return false, err
			}
			if !allowed {
				return false, nil
			}
		}
		return true, nil
	}
}

// OrGuard combines multiple guards with OR logic.
func OrGuard(guards ...Guard) Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		for _, guard := range guards {
			allowed, err := guard(ctx, event)
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
		return false, nil
	}
}

// NotGuard inverts a guard.
func NotGuard(guard Guard) Guard {
	return func(ctx context.Context, event Event) (bool, error) {
		allowed, err := guard(ctx, event)
		if err != nil {
			return false, err
		}
		return !allowed, nil
	}
}

// =============== Common Action Functions ===============

// NoOpAction is an action that does nothing.
func NoOpAction() Action {
	return func(ctx context.Context, from string, to string, event Event) error {
		return nil
	}
}

// LogAction creates an action that logs the transition.
func LogAction(logger func(msg string)) Action {
	return func(ctx context.Context, from string, to string, event Event) error {
		logger(fmt.Sprintf("Transition: %s -> %s (event: %s)", from, to, event.Name))
		return nil
	}
}

// ChainActions chains multiple actions together.
func ChainActions(actions ...Action) Action {
	return func(ctx context.Context, from string, to string, event Event) error {
		for _, action := range actions {
			if err := action(ctx, from, to, event); err != nil {
				return err
			}
		}
		return nil
	}
}
