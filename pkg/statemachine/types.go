// Package statemachine provides an event-driven state machine for Fluxor.
//
// The state machine integrates with Fluxor's EventBus for event-driven transitions,
// supports guards and actions, and provides observability through state change events.
//
// Example usage:
//
//	sm := statemachine.NewBuilder("order-processing").
//	    InitialState("pending").
//	    State("pending").
//	        On("approve", "approved").
//	        On("reject", "rejected").
//	        Entry(func(ctx context.Context, event Event) error {
//	            log.Println("Order is pending review")
//	            return nil
//	        }).
//	        Done().
//	    State("approved").
//	        On("ship", "shipped").
//	        Done().
//	    State("shipped").
//	        Final(true).
//	        Done().
//	    Build()
package statemachine

import (
	"context"
	"time"
)

// StateMachine defines the state machine interface.
type StateMachine interface {
	// ID returns the state machine instance ID
	ID() string

	// CurrentState returns the current state name
	CurrentState() string

	// Send sends an event to the state machine
	Send(ctx context.Context, event Event) error

	// SendAsync sends an event asynchronously and returns a Future
	SendAsync(ctx context.Context, event Event) Future

	// Start starts the state machine
	Start(ctx context.Context) error

	// Stop stops the state machine
	Stop(ctx context.Context) error

	// IsInState checks if machine is in the specified state
	IsInState(state string) bool

	// CanTransition checks if the event can trigger a transition from current state
	CanTransition(event string) bool

	// GetDefinition returns the state machine definition
	GetDefinition() *Definition

	// GetHistory returns the state transition history
	GetHistory() []HistoryEntry

	// Reset resets the state machine to initial state
	Reset(ctx context.Context) error
}

// Definition defines the complete state machine structure.
type Definition struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InitialState string                 `json:"initialState"`
	States       map[string]*State      `json:"states"`
	Context      map[string]interface{} `json:"context,omitempty"` // Shared context data
	Version      string                 `json:"version,omitempty"`
}

// State represents a single state in the state machine.
type State struct {
	Name        string                 `json:"name"`
	IsFinal     bool                   `json:"isFinal,omitempty"`
	Transitions []*Transition          `json:"transitions"`
	Entry       Handler                `json:"-"` // Called when entering state
	Exit        Handler                `json:"-"` // Called when exiting state
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Transition represents a transition between states.
type Transition struct {
	Event       string                 `json:"event"`         // Event that triggers transition
	From        string                 `json:"from"`          // Source state
	To          string                 `json:"to"`            // Target state
	Guard       Guard                  `json:"-"`             // Guard condition (must return true)
	Action      Action                 `json:"-"`             // Action to execute during transition
	Priority    int                    `json:"priority"`      // Higher priority transitions evaluated first
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Event represents a state machine event.
type Event struct {
	Name      string                 `json:"name"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"requestId,omitempty"`
}

// Handler is called on state entry or exit.
type Handler func(ctx context.Context, event Event) error

// Guard is a condition that must be satisfied for a transition to occur.
type Guard func(ctx context.Context, event Event) (bool, error)

// Action is executed during a state transition.
type Action func(ctx context.Context, from string, to string, event Event) error

// HistoryEntry records a state transition.
type HistoryEntry struct {
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"` // Time spent in previous state
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// StateChangeEvent is emitted when state changes.
type StateChangeEvent struct {
	MachineID string                 `json:"machineId"`
	From      string                 `json:"from"`
	To        string                 `json:"to"`
	Event     string                 `json:"event"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Future represents an async state transition result.
type Future interface {
	Await(ctx context.Context) error
	OnComplete(fn func(error))
	IsComplete() bool
}

// TransitionResult represents the result of a transition attempt.
type TransitionResult struct {
	Success       bool          `json:"success"`
	FromState     string        `json:"fromState"`
	ToState       string        `json:"toState"`
	Event         string        `json:"event"`
	Error         string        `json:"error,omitempty"`
	GuardRejected bool          `json:"guardRejected,omitempty"`
	Duration      time.Duration `json:"duration"`
}

// StateMachineError represents a state machine error.
type StateMachineError struct {
	Message   string
	Code      ErrorCode
	State     string
	Event     string
	Timestamp time.Time
}

func (e *StateMachineError) Error() string {
	return e.Message
}

// ErrorCode represents error types.
type ErrorCode int

const (
	ErrorCodeInvalidTransition ErrorCode = iota
	ErrorCodeGuardRejected
	ErrorCodeActionFailed
	ErrorCodeHandlerFailed
	ErrorCodeTimeout
	ErrorCodeInvalidState
	ErrorCodeMachineNotStarted
	ErrorCodeMachineStopped
)

// PersistenceAdapter defines the interface for state persistence.
type PersistenceAdapter interface {
	Save(ctx context.Context, machineID string, state string, context map[string]interface{}) error
	Load(ctx context.Context, machineID string) (state string, context map[string]interface{}, err error)
	Delete(ctx context.Context, machineID string) error
}

// Observer can observe state machine transitions.
type Observer interface {
	OnTransition(ctx context.Context, from string, to string, event Event)
	OnError(ctx context.Context, err error)
}
