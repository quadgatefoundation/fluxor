package fsm

import "context"

// StateConfigBuilder provides a fluent API for configuring states
type StateConfigBuilder struct {
	config *StateConfig
}

// Permit defines a allowed transition from the current state
func (b *StateConfigBuilder) Permit(event Event, nextState State) *StateConfigBuilder {
	return b.PermitIf(event, nextState, nil)
}

// PermitIf defines a allowed transition if the guard returns true
func (b *StateConfigBuilder) PermitIf(event Event, nextState State, guard Guard) *StateConfigBuilder {
	b.config.transitions[event] = &Transition{
		trigger: event,
		from:    b.config.state,
		to:      nextState,
		guard:   guard,
		actions: make([]Action, 0),
		kind:    TransitionExternal,
	}
	return b
}

// PermitWithAction defines a transition that executes an action
func (b *StateConfigBuilder) PermitWithAction(event Event, nextState State, action Action) *StateConfigBuilder {
	t := &Transition{
		trigger: event,
		from:    b.config.state,
		to:      nextState,
		actions: []Action{action},
		kind:    TransitionExternal,
	}
	b.config.transitions[event] = t
	return b
}

// Ignore defines an event that should be ignored (no transition, no error)
// For now, we don't explicitly track ignores, as Fire() errors on missing transition.
// TODO: Implement explicit Ignore support if needed (would return current state with no error).
func (b *StateConfigBuilder) Ignore(event Event) *StateConfigBuilder {
	// Implemented as internal transition with no action
	return b.InternalTransition(event, func(_ context.Context, _ TransitionContext) error {
		return nil
	})
}

// OnEntry adds an action to be executed when entering this state
func (b *StateConfigBuilder) OnEntry(action Action) *StateConfigBuilder {
	b.config.onEntry = append(b.config.onEntry, action)
	return b
}

// OnExit adds an action to be executed when exiting this state
func (b *StateConfigBuilder) OnExit(action Action) *StateConfigBuilder {
	b.config.onExit = append(b.config.onExit, action)
	return b
}

// InternalTransition defines a transition that executes an action but stays in the same state
// Importantly: OnEntry and OnExit handlers are NOT called
func (b *StateConfigBuilder) InternalTransition(event Event, action Action) *StateConfigBuilder {
	// Internal transition: to == from
	t := &Transition{
		trigger: event,
		from:    b.config.state,
		to:      b.config.state,
		actions: []Action{action},
		kind:    TransitionInternal,
	}
	b.config.transitions[event] = t
	return b
}
