package statemachine

import (
	"context"
)

// StateMachineBuilder provides a fluent API for building state machines.
type StateMachineBuilder struct {
	def *StateMachineDefinition
}

// NewStateMachineBuilder creates a new state machine builder.
func NewStateMachineBuilder(id, name string) *StateMachineBuilder {
	return &StateMachineBuilder{
		def: &StateMachineDefinition{
			ID:       id,
			Name:     name,
			States:   make([]StateDefinition, 0),
			Settings: make(map[string]interface{}),
		},
	}
}

// Description sets the description.
func (b *StateMachineBuilder) Description(desc string) *StateMachineBuilder {
	b.def.Description = desc
	return b
}

// Version sets the version.
func (b *StateMachineBuilder) Version(version string) *StateMachineBuilder {
	b.def.Version = version
	return b
}

// InitialState sets the initial state.
func (b *StateMachineBuilder) InitialState(stateID string) *StateMachineBuilder {
	b.def.InitialState = stateID
	return b
}

// AddState adds a state and returns a state builder.
func (b *StateMachineBuilder) AddState(id, name string) *StateBuilder {
	state := StateDefinition{
		ID:          id,
		Name:        name,
		Type:        StateTypeNormal,
		Transitions: make([]TransitionDefinition, 0),
		OnEnter:     make([]ActionDefinition, 0),
		OnExit:      make([]ActionDefinition, 0),
		Metadata:    make(map[string]interface{}),
	}
	return &StateBuilder{
		machineBuilder: b,
		state:          &state,
	}
}

// Setting sets a machine-level setting.
func (b *StateMachineBuilder) Setting(key string, value interface{}) *StateMachineBuilder {
	b.def.Settings[key] = value
	return b
}

// Build returns the completed state machine definition.
func (b *StateMachineBuilder) Build() *StateMachineDefinition {
	return b.def
}

// StateBuilder provides a fluent API for building states.
type StateBuilder struct {
	machineBuilder *StateMachineBuilder
	state          *StateDefinition
}

// Description sets the state description.
func (sb *StateBuilder) Description(desc string) *StateBuilder {
	sb.state.Description = desc
	return sb
}

// Type sets the state type.
func (sb *StateBuilder) Type(stateType StateType) *StateBuilder {
	sb.state.Type = stateType
	return sb
}

// Final marks the state as final.
func (sb *StateBuilder) Final(isFinal bool) *StateBuilder {
	if isFinal {
		sb.state.Type = StateTypeFinal
	}
	return sb
}

// Parent sets the parent state for hierarchical states.
func (sb *StateBuilder) Parent(parentID string) *StateBuilder {
	sb.state.Parent = parentID
	return sb
}

// AddTransition adds a transition and returns a transition builder.
func (sb *StateBuilder) AddTransition(event, target string) *TransitionBuilder {
	trans := TransitionDefinition{
		Event:   event,
		Target:  target,
		Actions: make([]ActionDefinition, 0),
	}
	return &TransitionBuilder{
		stateBuilder: sb,
		transition:   &trans,
	}
}

// OnEnter adds an onEnter action.
func (sb *StateBuilder) OnEnter(action ActionFunc) *StateBuilder {
	// Store as a function action with generated name
	actionDef := ActionDefinition{
		Type: "function",
		Name: "__generated_enter_" + sb.state.ID,
	}
	sb.state.OnEnter = append(sb.state.OnEnter, actionDef)
	return sb
}

// OnEnterAction adds a configured onEnter action.
func (sb *StateBuilder) OnEnterAction(actionDef ActionDefinition) *StateBuilder {
	sb.state.OnEnter = append(sb.state.OnEnter, actionDef)
	return sb
}

// OnExit adds an onExit action.
func (sb *StateBuilder) OnExit(action ActionFunc) *StateBuilder {
	// Store as a function action with generated name
	actionDef := ActionDefinition{
		Type: "function",
		Name: "__generated_exit_" + sb.state.ID,
	}
	sb.state.OnExit = append(sb.state.OnExit, actionDef)
	return sb
}

// OnExitAction adds a configured onExit action.
func (sb *StateBuilder) OnExitAction(actionDef ActionDefinition) *StateBuilder {
	sb.state.OnExit = append(sb.state.OnExit, actionDef)
	return sb
}

// Metadata sets metadata for the state.
func (sb *StateBuilder) Metadata(key string, value interface{}) *StateBuilder {
	sb.state.Metadata[key] = value
	return sb
}

// Done completes the state and returns to the machine builder.
func (sb *StateBuilder) Done() *StateMachineBuilder {
	sb.machineBuilder.def.States = append(sb.machineBuilder.def.States, *sb.state)
	return sb.machineBuilder
}

// TransitionBuilder provides a fluent API for building transitions.
type TransitionBuilder struct {
	stateBuilder *StateBuilder
	transition   *TransitionDefinition
}

// Guard sets a guard function name.
func (tb *TransitionBuilder) Guard(guardName string) *TransitionBuilder {
	tb.transition.Guard = guardName
	return tb
}

// GuardFunc sets an inline guard function (convenience method).
func (tb *TransitionBuilder) GuardFunc(guard GuardFunc) *TransitionBuilder {
	// Generate a unique name for this guard
	// The actual function will need to be registered separately
	tb.transition.Guard = "__generated_guard_" + tb.stateBuilder.state.ID + "_" + tb.transition.Event
	return tb
}

// GuardConfig sets guard configuration.
func (tb *TransitionBuilder) GuardConfig(config map[string]interface{}) *TransitionBuilder {
	tb.transition.GuardConfig = config
	return tb
}

// Action adds an action to the transition.
func (tb *TransitionBuilder) Action(actionName string) *TransitionBuilder {
	tb.transition.Actions = append(tb.transition.Actions, ActionDefinition{
		Type: "function",
		Name: actionName,
	})
	return tb
}

// ActionFunc adds an inline action function (convenience method).
func (tb *TransitionBuilder) ActionFunc(action ActionFunc) *TransitionBuilder {
	// Generate a unique name for this action
	// The actual function will need to be registered separately
	actionName := "__generated_action_" + tb.stateBuilder.state.ID + "_" + tb.transition.Event
	tb.transition.Actions = append(tb.transition.Actions, ActionDefinition{
		Type: "function",
		Name: actionName,
	})
	return tb
}

// ActionDef adds a configured action.
func (tb *TransitionBuilder) ActionDef(actionDef ActionDefinition) *TransitionBuilder {
	tb.transition.Actions = append(tb.transition.Actions, actionDef)
	return tb
}

// Priority sets the transition priority.
func (tb *TransitionBuilder) Priority(priority int) *TransitionBuilder {
	tb.transition.Priority = priority
	return tb
}

// Done completes the transition and returns to the state builder.
func (tb *TransitionBuilder) Done() *StateBuilder {
	tb.stateBuilder.state.Transitions = append(tb.stateBuilder.state.Transitions, *tb.transition)
	return tb.stateBuilder
}

// Helper functions for common action definitions

// PublishAction creates an EventBus publish action.
func PublishAction(address string) ActionDefinition {
	return ActionDefinition{
		Type: "eventbus",
		Config: map[string]interface{}{
			"address": address,
			"action":  "publish",
		},
	}
}

// SendAction creates an EventBus send action.
func SendAction(address string) ActionDefinition {
	return ActionDefinition{
		Type: "eventbus",
		Config: map[string]interface{}{
			"address": address,
			"action":  "send",
		},
	}
}

// SetVariableAction creates a set variable action.
func SetVariableAction(key string, value interface{}) ActionDefinition {
	return ActionDefinition{
		Type: "set",
		Config: map[string]interface{}{
			"values": map[string]interface{}{
				key: value,
			},
		},
	}
}

// FunctionAction creates a function action.
func FunctionAction(name string) ActionDefinition {
	return ActionDefinition{
		Type: "function",
		Name: name,
	}
}

// Helper guard builders

// AmountGreaterThanGuard creates a guard that checks if amount > value.
func AmountGreaterThanGuard(field string, value float64) GuardFunc {
	return func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		if event.Data == nil {
			return false
		}
		if amount, ok := event.Data[field].(float64); ok {
			return amount > value
		}
		return false
	}
}

// HasFieldGuard creates a guard that checks if a field exists.
func HasFieldGuard(field string) GuardFunc {
	return func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		if event.Data == nil {
			return false
		}
		_, ok := event.Data[field]
		return ok
	}
}

// EqualsGuard creates a guard that checks if a field equals a value.
func EqualsGuard(field string, value interface{}) GuardFunc {
	return func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool {
		if event.Data == nil {
			return false
		}
		return event.Data[field] == value
	}
}
