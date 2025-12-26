package statemachine

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// StateType represents the type of a state in the state machine.
type StateType string

// TransitionEvent represents an event that triggers a state transition.
type TransitionEvent string

// State represents a state in the state machine.
type State struct {
	// ID is the unique identifier for this state
	ID StateType

	// Name is a human-readable name for this state
	Name string

	// Description describes what this state represents
	Description string

	// IsFinal indicates if this is a terminal state
	IsFinal bool

	// OnEnter is called when entering this state
	OnEnter StateAction

	// OnExit is called when exiting this state
	OnExit StateAction

	// Metadata stores additional state-specific data
	Metadata map[string]interface{}
}

// Transition represents a transition between states.
type Transition struct {
	// ID is the unique identifier for this transition
	ID string

	// From is the source state
	From StateType

	// To is the target state
	To StateType

	// Event is the event that triggers this transition
	Event TransitionEvent

	// Guard is an optional condition that must be true for transition to occur
	Guard TransitionGuard

	// Action is executed during the transition
	Action TransitionAction

	// Priority determines the order of evaluation (higher = evaluated first)
	Priority int

	// Metadata stores additional transition-specific data
	Metadata map[string]interface{}
}

// StateAction is a function executed on state entry/exit.
type StateAction func(ctx *StateContext) error

// TransitionAction is a function executed during a transition.
type TransitionAction func(ctx *StateContext, event *Event) error

// TransitionGuard is a function that determines if a transition can occur.
type TransitionGuard func(ctx *StateContext, event *Event) (bool, error)

// StateContext provides context for state machine operations.
type StateContext struct {
	// MachineID is the unique identifier for this state machine instance
	MachineID string

	// CurrentState is the current state
	CurrentState StateType

	// PreviousState is the previous state (if any)
	PreviousState *StateType

	// Data stores state machine instance data
	Data map[string]interface{}

	// Context is the Go context for cancellation/timeout
	Context context.Context

	// FluxorContext provides access to Vertx/EventBus
	FluxorContext core.FluxorContext

	// History tracks state transitions
	History []*HistoryEntry

	// StartTime is when this state machine instance was created
	StartTime time.Time

	// LastTransitionTime is when the last transition occurred
	LastTransitionTime time.Time
}

// Event represents an event that may trigger a transition.
type Event struct {
	// Name is the event name/type
	Name TransitionEvent

	// Data contains event-specific data
	Data map[string]interface{}

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Source identifies where the event came from
	Source string

	// RequestID for tracing
	RequestID string
}

// HistoryEntry records a state transition.
type HistoryEntry struct {
	// FromState is the source state
	FromState StateType

	// ToState is the target state
	ToState StateType

	// Event is the triggering event
	Event TransitionEvent

	// Timestamp is when the transition occurred
	Timestamp time.Time

	// TransitionID identifies which transition was used
	TransitionID string

	// Data is a snapshot of state machine data at transition time
	Data map[string]interface{}
}

// StateMachineDefinition defines a state machine.
type StateMachineDefinition struct {
	// ID is the unique identifier for this state machine definition
	ID string

	// Name is a human-readable name
	Name string

	// Description describes what this state machine does
	Description string

	// InitialState is the starting state
	InitialState StateType

	// States defines all possible states
	States map[StateType]*State

	// Transitions defines all possible transitions
	Transitions []*Transition

	// Version for schema evolution
	Version string

	// Metadata stores additional definition data
	Metadata map[string]interface{}
}

// StateMachineInstance represents a running instance of a state machine.
type StateMachineInstance struct {
	// ID is the unique identifier for this instance
	ID string

	// DefinitionID references the state machine definition
	DefinitionID string

	// Context is the current state context
	Context *StateContext

	// Status tracks the instance lifecycle
	Status InstanceStatus

	// CreatedAt is when the instance was created
	CreatedAt time.Time

	// UpdatedAt is when the instance was last updated
	UpdatedAt time.Time

	// CompletedAt is when the instance reached a final state (if any)
	CompletedAt *time.Time
}

// InstanceStatus represents the lifecycle status of a state machine instance.
type InstanceStatus string

const (
	// InstanceStatusActive means the instance is active and can process events
	InstanceStatusActive InstanceStatus = "active"

	// InstanceStatusCompleted means the instance reached a final state
	InstanceStatusCompleted InstanceStatus = "completed"

	// InstanceStatusFailed means the instance encountered an error
	InstanceStatusFailed InstanceStatus = "failed"

	// InstanceStatusSuspended means the instance is paused
	InstanceStatusSuspended InstanceStatus = "suspended"
)

// TransitionResult represents the result of a state transition.
type TransitionResult struct {
	// Success indicates if the transition was successful
	Success bool

	// FromState is the state before transition
	FromState StateType

	// ToState is the state after transition
	ToState StateType

	// Event is the event that triggered the transition
	Event TransitionEvent

	// Error is set if the transition failed
	Error error

	// TransitionID identifies which transition was used
	TransitionID string

	// Timestamp is when the transition occurred
	Timestamp time.Time
}

// StateMachineConfig configures a state machine engine.
type StateMachineConfig struct {
	// EnableHistory enables transition history tracking
	EnableHistory bool

	// MaxHistorySize limits the number of history entries (0 = unlimited)
	MaxHistorySize int

	// EnablePersistence enables state persistence
	EnablePersistence bool

	// PersistenceStore is the storage backend
	PersistenceStore PersistenceStore

	// EnableEventBus enables EventBus integration
	EnableEventBus bool

	// EventBusPrefix is the prefix for EventBus addresses
	EventBusPrefix string

	// DefaultTimeout for state actions and transitions
	DefaultTimeout time.Duration
}

// PersistenceStore provides storage for state machine instances.
type PersistenceStore interface {
	// Save persists a state machine instance
	Save(ctx context.Context, instance *StateMachineInstance) error

	// Load retrieves a state machine instance by ID
	Load(ctx context.Context, instanceID string) (*StateMachineInstance, error)

	// Delete removes a state machine instance
	Delete(ctx context.Context, instanceID string) error

	// List returns all instances for a given definition
	List(ctx context.Context, definitionID string) ([]*StateMachineInstance, error)
}

// StateMachine is the main interface for interacting with state machines.
type StateMachine interface {
	// GetDefinition returns the state machine definition
	GetDefinition() *StateMachineDefinition

	// CreateInstance creates a new state machine instance
	CreateInstance(ctx context.Context, initialData map[string]interface{}) (*StateMachineInstance, error)

	// SendEvent sends an event to a state machine instance
	SendEvent(ctx context.Context, instanceID string, event *Event) (*TransitionResult, error)

	// GetInstance retrieves a state machine instance
	GetInstance(ctx context.Context, instanceID string) (*StateMachineInstance, error)

	// GetCurrentState returns the current state of an instance
	GetCurrentState(ctx context.Context, instanceID string) (StateType, error)

	// CanTransition checks if a transition is possible for a given event
	CanTransition(ctx context.Context, instanceID string, event *Event) (bool, error)

	// Stop stops the state machine engine
	Stop() error
}
