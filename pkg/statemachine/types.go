// Package statemachine provides a state machine implementation for Fluxor
// using EventBus for state transitions and event-driven execution.
//
// Features:
// - Declarative state definitions (JSON-serializable)
// - Event-driven transitions via EventBus
// - Guards (conditions) for transitions
// - Actions (enter/exit/transition handlers)
// - Hierarchical/nested states
// - State persistence and recovery
// - Observable state changes
// - Future/Promise integration for async operations
//
// Example:
//
//	machine := statemachine.NewStateMachineBuilder("order-fsm", "Order State Machine").
//		InitialState("created").
//		AddState("created", "Order Created").
//			AddTransition("process", "processing").
//				Guard(func(ctx context.Context, event *Event) bool {
//					return event.Data["amount"].(float64) > 0
//				}).
//				Action(func(ctx context.Context, event *Event) error {
//					// Validate order
//					return nil
//				}).
//				Done().
//			Done().
//		AddState("processing", "Processing Order").
//			OnEnter(func(ctx context.Context, event *Event) error {
//				// Start processing
//				return nil
//			}).
//			AddTransition("complete", "completed").Done().
//			AddTransition("fail", "failed").Done().
//			Done().
//		AddState("completed", "Order Completed").Final(true).Done().
//		AddState("failed", "Order Failed").Final(true).Done().
//		Build()
package statemachine

import (
	"context"
	"time"
)

// StateMachineDefinition defines a finite state machine in JSON-serializable format.
type StateMachineDefinition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Version      string                 `json:"version,omitempty"`
	InitialState string                 `json:"initialState"`
	States       []StateDefinition      `json:"states"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// StateDefinition defines a single state in the state machine.
type StateDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        StateType              `json:"type,omitempty"` // normal, initial, final, parallel
	Transitions []TransitionDefinition `json:"transitions"`
	OnEnter     []ActionDefinition     `json:"onEnter,omitempty"`
	OnExit      []ActionDefinition     `json:"onExit,omitempty"`
	Parent      string                 `json:"parent,omitempty"`      // For hierarchical states
	SubStates   []string               `json:"subStates,omitempty"`   // For composite states
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TransitionDefinition defines a transition between states.
type TransitionDefinition struct {
	Event       string                 `json:"event"`                 // Event that triggers transition
	Target      string                 `json:"target"`                // Target state ID
	Guard       string                 `json:"guard,omitempty"`       // Guard function name
	GuardConfig map[string]interface{} `json:"guardConfig,omitempty"` // Guard configuration
	Actions     []ActionDefinition     `json:"actions,omitempty"`     // Actions to execute
	Priority    int                    `json:"priority,omitempty"`    // Higher priority evaluated first
}

// ActionDefinition defines an action to execute.
type ActionDefinition struct {
	Type   string                 `json:"type"`             // function, eventbus, http, code
	Name   string                 `json:"name,omitempty"`   // Action name (for function type)
	Config map[string]interface{} `json:"config,omitempty"` // Action configuration
}

// StateType represents the type of state.
type StateType string

const (
	StateTypeNormal   StateType = "normal"   // Regular state
	StateTypeInitial  StateType = "initial"  // Initial state (automatically entered)
	StateTypeFinal    StateType = "final"    // Final state (machine stops)
	StateTypeParallel StateType = "parallel" // Parallel state (multiple active substates)
)

// ExecutionContext holds the context for a state machine execution (instance).
type ExecutionContext struct {
	MachineID    string                 `json:"machineId"`
	InstanceID   string                 `json:"instanceId"`
	CurrentState string                 `json:"currentState"`
	PreviousState string                `json:"previousState,omitempty"`
	StartTime    time.Time              `json:"startTime"`
	UpdateTime   time.Time              `json:"updateTime"`
	Data         map[string]interface{} `json:"data"`         // Instance data
	Variables    map[string]interface{} `json:"variables"`    // User-defined variables
	History      []StateTransition      `json:"history"`      // Transition history
	Status       ExecutionStatus        `json:"status"`
	ActiveStates []string               `json:"activeStates,omitempty"` // For parallel states
}

// StateTransition records a single state transition.
type StateTransition struct {
	FromState string                 `json:"fromState"`
	ToState   string                 `json:"toState"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ExecutionStatus represents the status of a state machine instance.
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusSuspended ExecutionStatus = "suspended"
)

// Event represents an event that triggers a state transition.
type Event struct {
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// GuardFunc is a function that determines if a transition can occur.
// Returns true if the transition should proceed, false otherwise.
type GuardFunc func(ctx context.Context, event *Event, execCtx *ExecutionContext) bool

// ActionFunc is a function executed during state transitions or state enter/exit.
// Returns an error if the action fails.
type ActionFunc func(ctx context.Context, event *Event, execCtx *ExecutionContext) error

// StateHandler handles state-specific logic.
type StateHandler interface {
	OnEnter(ctx context.Context, event *Event, execCtx *ExecutionContext) error
	OnExit(ctx context.Context, event *Event, execCtx *ExecutionContext) error
}

// StateMachineEngine manages state machine execution.
type StateMachineEngine interface {
	// RegisterMachine registers a state machine definition
	RegisterMachine(def *StateMachineDefinition) error

	// CreateInstance creates a new state machine instance
	CreateInstance(ctx context.Context, machineID string, initialData map[string]interface{}) (string, error)

	// SendEvent sends an event to a state machine instance
	SendEvent(ctx context.Context, instanceID string, event *Event) error

	// GetInstance returns the execution context for an instance
	GetInstance(instanceID string) (*ExecutionContext, error)

	// ListInstances lists all instances for a machine
	ListInstances(machineID string) ([]*ExecutionContext, error)

	// SaveInstance persists instance state (for recovery)
	SaveInstance(instanceID string) error

	// RestoreInstance restores instance state from persistence
	RestoreInstance(instanceID string) error
}

// StateMachineVerticle is a deployable unit for state machines.
type StateMachineVerticle interface {
	// Engine returns the underlying state machine engine
	Engine() StateMachineEngine

	// RegisterGuard registers a guard function
	RegisterGuard(name string, guard GuardFunc)

	// RegisterAction registers an action function
	RegisterAction(name string, action ActionFunc)

	// RegisterStateHandler registers a state handler
	RegisterStateHandler(stateID string, handler StateHandler)
}

// StateChangeListener is notified of state changes.
type StateChangeListener func(ctx context.Context, instanceID string, from, to string, event *Event)

// PersistenceProvider handles state machine persistence.
type PersistenceProvider interface {
	Save(instanceID string, execCtx *ExecutionContext) error
	Load(instanceID string) (*ExecutionContext, error)
	Delete(instanceID string) error
	List(machineID string) ([]string, error)
}
