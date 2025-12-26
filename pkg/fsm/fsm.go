package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/fluxor"
)

// State represents a state identifier (generic string for flexibility)
type State string

// Event represents an event identifier
type Event string

// Action is a function executed during transitions
// Returns an error which can stop the transition or be logged
type Action func(ctx context.Context, transition TransitionContext) error

// Guard is a function that decides if a transition can occur
type Guard func(ctx context.Context, transition TransitionContext) bool

// TransitionType defines the type of transition
type TransitionType int

const (
	// TransitionExternal causes a state change (exits source, enters target)
	TransitionExternal TransitionType = iota
	// TransitionInternal does not cause a state change (no exit/entry)
	TransitionInternal
)

// TransitionContext holds context about the current transition
type TransitionContext struct {
	FSM       *StateMachine
	Event     Event
	From      State
	To        State
	Data      any
}

// StateMachine implements a Finite State Machine
// Thread-safe and reactive
type StateMachine struct {
	id           string
	currentState State
	states       map[State]*StateConfig
	mu           sync.RWMutex
	
	// Global interceptors
	onTransition []func(TransitionContext)
}

// StateConfig represents the configuration for a specific state
type StateConfig struct {
	state        State
	onEntry      []Action
	onExit       []Action
	transitions  map[Event]*Transition
	parent       *StateConfig // For hierarchical states (future)
}

// Transition represents a state transition definition
type Transition struct {
	trigger Event
	from    State
	to      State
	guard   Guard
	actions []Action
	kind    TransitionType
}

// New creates a new StateMachine with an initial state
func New(id string, initialState State) *StateMachine {
	return &StateMachine{
		id:           id,
		currentState: initialState,
		states:       make(map[State]*StateConfig),
		onTransition: make([]func(TransitionContext), 0),
	}
}

// CurrentState returns the current state
func (sm *StateMachine) CurrentState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// Configure returns a StateConfigBuilder for the given state
// If the state config doesn't exist, it creates one
func (sm *StateMachine) Configure(state State) *StateConfigBuilder {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	config, ok := sm.states[state]
	if !ok {
		config = &StateConfig{
			state:       state,
			onEntry:     make([]Action, 0),
			onExit:      make([]Action, 0),
			transitions: make(map[Event]*Transition),
		}
		sm.states[state] = config
	}

	return &StateConfigBuilder{
		config: config,
	}
}

// Fire triggers an event and returns a FutureT[State]
// The future completes with the new state, or fails with an error
func (sm *StateMachine) Fire(ctx context.Context, event Event, data any) *fluxor.FutureT[State] {
	promise := fluxor.NewPromiseT[State]()

	// Execute in goroutine to be non-blocking/reactive
	go func() {
		sm.mu.Lock()
		defer sm.mu.Unlock()

		currentState := sm.currentState
		stateConfig, ok := sm.states[currentState]
		
		// If no config for current state, cannot transition
		if !ok {
			promise.Fail(fmt.Errorf("no configuration for state %s", currentState))
			return
		}

		// Check if transition exists for event
		transition, ok := stateConfig.transitions[event]
		if !ok {
			promise.Fail(fmt.Errorf("no transition defined for event %s in state %s", event, currentState))
			return
		}

		// Create transition context
		tCtx := TransitionContext{
			FSM:   sm,
			Event: event,
			From:  currentState,
			To:    transition.to,
			Data:  data,
		}

		// Check guard if present
		if transition.guard != nil {
			if !transition.guard(ctx, tCtx) {
				promise.Fail(fmt.Errorf("guard failed for transition %s -> %s on event %s", currentState, transition.to, event))
				return
			}
		}

		// 1. Execute Exit actions of current state (ONLY if External)
		if transition.kind == TransitionExternal {
			for _, action := range stateConfig.onExit {
				if err := action(ctx, tCtx); err != nil {
					promise.Fail(fmt.Errorf("exit action failed: %w", err))
					return
				}
			}
		}

		// 2. Execute Transition actions
		for _, action := range transition.actions {
			if err := action(ctx, tCtx); err != nil {
				promise.Fail(fmt.Errorf("transition action failed: %w", err))
				return
			}
		}

		// 3. Update State
		sm.currentState = transition.to

		// 4. Execute Entry actions of new state (ONLY if External)
		if transition.kind == TransitionExternal {
			newStateConfig, ok := sm.states[transition.to]
			if ok {
				for _, action := range newStateConfig.onEntry {
					if err := action(ctx, tCtx); err != nil {
						// State is already updated, but entry failed
						promise.Fail(fmt.Errorf("entry action failed: %w", err))
						return
					}
				}
			}
		}

		// Notify global listeners
		for _, listener := range sm.onTransition {
			listener(tCtx)
		}

		promise.Complete(sm.currentState)
	}()

	return &promise.FutureT
}

// OnTransition registers a global transition listener
func (sm *StateMachine) OnTransition(listener func(TransitionContext)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onTransition = append(sm.onTransition, listener)
}
