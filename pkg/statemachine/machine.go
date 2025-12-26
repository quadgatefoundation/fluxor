package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/google/uuid"
)

// stateMachine implements StateMachine.
type stateMachine struct {
	id         string
	definition *Definition
	current    string
	context    map[string]interface{}
	history    []HistoryEntry
	observers  []Observer
	
	// State tracking
	started      bool
	stopped      bool
	stateEnteredAt time.Time
	
	// Concurrency control
	mu           sync.RWMutex
	transitionMu sync.Mutex // Ensures only one transition at a time
	
	// EventBus integration
	eventBus   core.EventBus
	logger     core.Logger
	
	// Persistence
	persistence PersistenceAdapter
}

// NewStateMachine creates a new state machine instance.
func NewStateMachine(def *Definition, opts ...Option) (StateMachine, error) {
	if err := validateDefinition(def); err != nil {
		return nil, fmt.Errorf("invalid definition: %w", err)
	}

	sm := &stateMachine{
		id:         uuid.New().String(),
		definition: def,
		current:    def.InitialState,
		context:    make(map[string]interface{}),
		history:    make([]HistoryEntry, 0),
		observers:  make([]Observer, 0),
		logger:     core.NewDefaultLogger(),
		stateEnteredAt: time.Now(),
	}

	// Copy context from definition
	if def.Context != nil {
		for k, v := range def.Context {
			sm.context[k] = v
		}
	}

	// Apply options
	for _, opt := range opts {
		opt(sm)
	}

	return sm, nil
}

// Option configures a state machine.
type Option func(*stateMachine)

// WithID sets a custom state machine ID.
func WithID(id string) Option {
	return func(sm *stateMachine) {
		sm.id = id
	}
}

// WithEventBus enables EventBus integration.
func WithEventBus(eb core.EventBus) Option {
	return func(sm *stateMachine) {
		sm.eventBus = eb
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger core.Logger) Option {
	return func(sm *stateMachine) {
		sm.logger = logger
	}
}

// WithPersistence enables state persistence.
func WithPersistence(adapter PersistenceAdapter) Option {
	return func(sm *stateMachine) {
		sm.persistence = adapter
	}
}

// WithObserver adds an observer.
func WithObserver(observer Observer) Option {
	return func(sm *stateMachine) {
		sm.observers = append(sm.observers, observer)
	}
}

// WithInitialContext sets initial context data.
func WithInitialContext(ctx map[string]interface{}) Option {
	return func(sm *stateMachine) {
		for k, v := range ctx {
			sm.context[k] = v
		}
	}
}

// ID returns the state machine instance ID.
func (sm *stateMachine) ID() string {
	return sm.id
}

// CurrentState returns the current state name.
func (sm *stateMachine) CurrentState() string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// Start starts the state machine.
func (sm *stateMachine) Start(ctx context.Context) error {
	sm.mu.Lock()
	if sm.started {
		sm.mu.Unlock()
		return fmt.Errorf("state machine already started")
	}
	sm.started = true
	sm.mu.Unlock()

	// Load persisted state if available
	if sm.persistence != nil {
		state, persistedCtx, err := sm.persistence.Load(ctx, sm.id)
		if err == nil && state != "" {
			sm.mu.Lock()
			sm.current = state
			if persistedCtx != nil {
				sm.context = persistedCtx
			}
			sm.mu.Unlock()
			sm.logger.Infof("State machine %s restored to state %s", sm.id, state)
		}
	}

	// Execute entry handler for initial state
	initialState := sm.getState(sm.current)
	if initialState != nil && initialState.Entry != nil {
		event := Event{
			Name:      "__start__",
			Timestamp: time.Now(),
		}
		if err := initialState.Entry(ctx, event); err != nil {
			sm.logger.Errorf("Initial state entry handler failed: %v", err)
			sm.notifyError(ctx, err)
		}
	}

	sm.logger.Infof("State machine %s started in state %s", sm.id, sm.current)

	// Register EventBus consumer if EventBus is configured
	if sm.eventBus != nil {
		sm.registerEventBusConsumer()
	}

	return nil
}

// Stop stops the state machine.
func (sm *stateMachine) Stop(ctx context.Context) error {
	sm.mu.Lock()
	if sm.stopped {
		sm.mu.Unlock()
		return fmt.Errorf("state machine already stopped")
	}
	sm.stopped = true
	sm.mu.Unlock()

	sm.logger.Infof("State machine %s stopped in state %s", sm.id, sm.current)
	return nil
}

// Send sends an event to the state machine.
func (sm *stateMachine) Send(ctx context.Context, event Event) error {
	sm.mu.RLock()
	if !sm.started || sm.stopped {
		sm.mu.RUnlock()
		return &StateMachineError{
			Message:   "state machine not started or stopped",
			Code:      ErrorCodeMachineNotStarted,
			Timestamp: time.Now(),
		}
	}
	sm.mu.RUnlock()

	// Ensure only one transition at a time
	sm.transitionMu.Lock()
	defer sm.transitionMu.Unlock()

	return sm.processEvent(ctx, event)
}

// SendAsync sends an event asynchronously.
func (sm *stateMachine) SendAsync(ctx context.Context, event Event) Future {
	future := newFuture()
	
	go func() {
		err := sm.Send(ctx, event)
		future.complete(err)
	}()
	
	return future
}

// processEvent processes an event and performs state transition.
func (sm *stateMachine) processEvent(ctx context.Context, event Event) error {
	startTime := time.Now()
	
	sm.mu.RLock()
	currentState := sm.current
	sm.mu.RUnlock()

	// Find matching transition
	transition := sm.findTransition(currentState, event.Name)
	if transition == nil {
		return &StateMachineError{
			Message:   fmt.Sprintf("no transition for event '%s' from state '%s'", event.Name, currentState),
			Code:      ErrorCodeInvalidTransition,
			State:     currentState,
			Event:     event.Name,
			Timestamp: time.Now(),
		}
	}

	// Evaluate guard
	if transition.Guard != nil {
		allowed, err := transition.Guard(ctx, event)
		if err != nil {
			sm.logger.Errorf("Guard evaluation failed: %v", err)
			return &StateMachineError{
				Message:   fmt.Sprintf("guard evaluation failed: %v", err),
				Code:      ErrorCodeGuardRejected,
				State:     currentState,
				Event:     event.Name,
				Timestamp: time.Now(),
			}
		}
		if !allowed {
			sm.logger.Debugf("Guard rejected transition from %s to %s", currentState, transition.To)
			return &StateMachineError{
				Message:   fmt.Sprintf("guard rejected transition"),
				Code:      ErrorCodeGuardRejected,
				State:     currentState,
				Event:     event.Name,
				Timestamp: time.Now(),
			}
		}
	}

	// Execute exit handler of current state
	currentStateObj := sm.getState(currentState)
	if currentStateObj != nil && currentStateObj.Exit != nil {
		if err := currentStateObj.Exit(ctx, event); err != nil {
			sm.logger.Errorf("Exit handler failed: %v", err)
			sm.notifyError(ctx, err)
			return &StateMachineError{
				Message:   fmt.Sprintf("exit handler failed: %v", err),
				Code:      ErrorCodeHandlerFailed,
				State:     currentState,
				Event:     event.Name,
				Timestamp: time.Now(),
			}
		}
	}

	// Execute transition action
	if transition.Action != nil {
		actionCtx := ctx
		if transition.Timeout > 0 {
			var cancel context.CancelFunc
			actionCtx, cancel = context.WithTimeout(ctx, transition.Timeout)
			defer cancel()
		}
		
		if err := transition.Action(actionCtx, currentState, transition.To, event); err != nil {
			sm.logger.Errorf("Transition action failed: %v", err)
			sm.notifyError(ctx, err)
			return &StateMachineError{
				Message:   fmt.Sprintf("transition action failed: %v", err),
				Code:      ErrorCodeActionFailed,
				State:     currentState,
				Event:     event.Name,
				Timestamp: time.Now(),
			}
		}
	}

	// Update state
	sm.mu.Lock()
	previousState := sm.current
	sm.current = transition.To
	duration := time.Since(sm.stateEnteredAt)
	sm.stateEnteredAt = time.Now()
	sm.mu.Unlock()

	// Record history
	historyEntry := HistoryEntry{
		From:      previousState,
		To:        transition.To,
		Event:     event.Name,
		Timestamp: time.Now(),
		Duration:  duration,
		Data:      event.Data,
	}
	sm.mu.Lock()
	sm.history = append(sm.history, historyEntry)
	sm.mu.Unlock()

	// Execute entry handler of new state
	newStateObj := sm.getState(transition.To)
	if newStateObj != nil && newStateObj.Entry != nil {
		if err := newStateObj.Entry(ctx, event); err != nil {
			sm.logger.Errorf("Entry handler failed: %v", err)
			sm.notifyError(ctx, err)
			// Don't return error - state has already changed
		}
	}

	// Persist state if enabled
	if sm.persistence != nil {
		if err := sm.persistence.Save(ctx, sm.id, sm.current, sm.context); err != nil {
			sm.logger.Errorf("Failed to persist state: %v", err)
		}
	}

	// Notify observers
	sm.notifyObservers(ctx, previousState, transition.To, event)

	// Publish state change event to EventBus
	if sm.eventBus != nil {
		changeEvent := StateChangeEvent{
			MachineID: sm.id,
			From:      previousState,
			To:        transition.To,
			Event:     event.Name,
			Timestamp: time.Now(),
			Data:      event.Data,
		}
		address := fmt.Sprintf("statemachine.%s.transition", sm.id)
		if err := sm.eventBus.Publish(address, changeEvent); err != nil {
			sm.logger.Errorf("Failed to publish state change event: %v", err)
		}
	}

	sm.logger.Infof("State machine %s transitioned from %s to %s (event: %s, duration: %v)", 
		sm.id, previousState, transition.To, event.Name, time.Since(startTime))

	return nil
}

// findTransition finds a matching transition for the given state and event.
func (sm *stateMachine) findTransition(state string, event string) *Transition {
	stateObj := sm.getState(state)
	if stateObj == nil {
		return nil
	}

	// Sort transitions by priority (higher first)
	var best *Transition
	for _, t := range stateObj.Transitions {
		if t.Event == event {
			if best == nil || t.Priority > best.Priority {
				best = t
			}
		}
	}

	return best
}

// getState gets a state definition by name.
func (sm *stateMachine) getState(name string) *State {
	return sm.definition.States[name]
}

// IsInState checks if machine is in the specified state.
func (sm *stateMachine) IsInState(state string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current == state
}

// CanTransition checks if the event can trigger a transition from current state.
func (sm *stateMachine) CanTransition(event string) bool {
	sm.mu.RLock()
	current := sm.current
	sm.mu.RUnlock()
	
	return sm.findTransition(current, event) != nil
}

// GetDefinition returns the state machine definition.
func (sm *stateMachine) GetDefinition() *Definition {
	return sm.definition
}

// GetHistory returns the state transition history.
func (sm *stateMachine) GetHistory() []HistoryEntry {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Return a copy
	history := make([]HistoryEntry, len(sm.history))
	copy(history, sm.history)
	return history
}

// Reset resets the state machine to initial state.
func (sm *stateMachine) Reset(ctx context.Context) error {
	sm.transitionMu.Lock()
	defer sm.transitionMu.Unlock()

	sm.mu.Lock()
	oldState := sm.current
	sm.current = sm.definition.InitialState
	sm.history = make([]HistoryEntry, 0)
	sm.stateEnteredAt = time.Now()
	
	// Clear context and restore from definition
	sm.context = make(map[string]interface{})
	if sm.definition.Context != nil {
		for k, v := range sm.definition.Context {
			sm.context[k] = v
		}
	}
	sm.mu.Unlock()

	sm.logger.Infof("State machine %s reset from %s to %s", sm.id, oldState, sm.current)

	// Clear persistence
	if sm.persistence != nil {
		if err := sm.persistence.Delete(ctx, sm.id); err != nil {
			sm.logger.Errorf("Failed to clear persisted state: %v", err)
		}
	}

	return nil
}

// notifyObservers notifies all observers of a transition.
func (sm *stateMachine) notifyObservers(ctx context.Context, from string, to string, event Event) {
	for _, observer := range sm.observers {
		go observer.OnTransition(ctx, from, to, event)
	}
}

// notifyError notifies all observers of an error.
func (sm *stateMachine) notifyError(ctx context.Context, err error) {
	for _, observer := range sm.observers {
		go observer.OnError(ctx, err)
	}
}

// registerEventBusConsumer registers an EventBus consumer for receiving events.
func (sm *stateMachine) registerEventBusConsumer() {
	address := fmt.Sprintf("statemachine.%s.event", sm.id)
	sm.eventBus.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var event Event
		if body, ok := msg.Body().([]byte); ok {
			if err := core.JSONDecode(body, &event); err != nil {
				return fmt.Errorf("failed to decode event: %w", err)
			}
		} else {
			// Try to convert directly
			event = Event{
				Name:      fmt.Sprintf("%v", msg.Body()),
				Timestamp: time.Now(),
			}
		}

		if err := sm.Send(ctx.Context(), event); err != nil {
			return msg.Reply(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}

		return msg.Reply(map[string]interface{}{
			"success": true,
			"state":   sm.current,
		})
	})
}

// validateDefinition validates a state machine definition.
func validateDefinition(def *Definition) error {
	if def.ID == "" {
		return fmt.Errorf("definition ID is required")
	}
	if def.InitialState == "" {
		return fmt.Errorf("initial state is required")
	}
	if len(def.States) == 0 {
		return fmt.Errorf("at least one state is required")
	}
	if _, ok := def.States[def.InitialState]; !ok {
		return fmt.Errorf("initial state '%s' not found in states", def.InitialState)
	}

	// Validate transitions
	for stateName, state := range def.States {
		for _, transition := range state.Transitions {
			if transition.Event == "" {
				return fmt.Errorf("transition event is required in state '%s'", stateName)
			}
			if transition.To == "" {
				return fmt.Errorf("transition target is required in state '%s'", stateName)
			}
			if _, ok := def.States[transition.To]; !ok {
				return fmt.Errorf("transition target '%s' not found in states (from state '%s')", transition.To, stateName)
			}
		}
	}

	return nil
}

// future implements Future.
type future struct {
	done     bool
	err      error
	mu       sync.Mutex
	callback func(error)
}

func newFuture() *future {
	return &future{}
}

func (f *future) complete(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.done {
		return
	}
	
	f.done = true
	f.err = err
	
	if f.callback != nil {
		f.callback(err)
	}
}

func (f *future) Await(ctx context.Context) error {
	for {
		f.mu.Lock()
		if f.done {
			err := f.err
			f.mu.Unlock()
			return err
		}
		f.mu.Unlock()
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			// Continue checking
		}
	}
}

func (f *future) OnComplete(fn func(error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.done {
		fn(f.err)
	} else {
		f.callback = fn
	}
}

func (f *future) IsComplete() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.done
}
