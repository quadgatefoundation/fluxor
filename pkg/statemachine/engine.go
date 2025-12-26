package statemachine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/google/uuid"
)

// Engine implements the StateMachine interface.
type Engine struct {
	definition *StateMachineDefinition
	config     *StateMachineConfig
	instances  map[string]*StateMachineInstance
	mu         sync.RWMutex
	eventBus   core.EventBus
	logger     core.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewEngine creates a new state machine engine.
func NewEngine(definition *StateMachineDefinition, config *StateMachineConfig, eventBus core.EventBus) (*Engine, error) {
	if err := validateDefinition(definition); err != nil {
		return nil, fmt.Errorf("invalid state machine definition: %w", err)
	}

	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		definition: definition,
		config:     config,
		instances:  make(map[string]*StateMachineInstance),
		eventBus:   eventBus,
		logger:     core.NewDefaultLogger(),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Register EventBus consumers if enabled
	if config.EnableEventBus && eventBus != nil {
		engine.registerEventBusConsumers()
	}

	return engine, nil
}

// GetDefinition returns the state machine definition.
func (e *Engine) GetDefinition() *StateMachineDefinition {
	return e.definition
}

// CreateInstance creates a new state machine instance.
func (e *Engine) CreateInstance(ctx context.Context, initialData map[string]interface{}) (*StateMachineInstance, error) {
	instanceID := uuid.New().String()

	if initialData == nil {
		initialData = make(map[string]interface{})
	}

	now := time.Now()
	stateCtx := &StateContext{
		MachineID:          instanceID,
		CurrentState:       e.definition.InitialState,
		PreviousState:      nil,
		Data:               initialData,
		Context:            ctx,
		History:            make([]*HistoryEntry, 0),
		StartTime:          now,
		LastTransitionTime: now,
	}

	instance := &StateMachineInstance{
		ID:           instanceID,
		DefinitionID: e.definition.ID,
		Context:      stateCtx,
		Status:       InstanceStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Store instance
	e.mu.Lock()
	e.instances[instanceID] = instance
	e.mu.Unlock()

	// Persist if enabled
	if e.config.EnablePersistence && e.config.PersistenceStore != nil {
		if err := e.config.PersistenceStore.Save(ctx, instance); err != nil {
			e.logger.Errorf("Failed to persist state machine instance %s: %v", instanceID, err)
		}
	}

	// Execute OnEnter for initial state
	initialState := e.definition.States[e.definition.InitialState]
	if initialState != nil && initialState.OnEnter != nil {
		if err := initialState.OnEnter(stateCtx); err != nil {
			e.logger.Errorf("Error executing OnEnter for initial state %s: %v", e.definition.InitialState, err)
			return nil, fmt.Errorf("failed to enter initial state: %w", err)
		}
	}

	// Publish instance created event
	if e.config.EnableEventBus && e.eventBus != nil {
		e.publishEvent("instance.created", map[string]interface{}{
			"machineId":     instanceID,
			"definitionId":  e.definition.ID,
			"initialState":  e.definition.InitialState,
			"timestamp":     now,
		})
	}

	e.logger.Infof("Created state machine instance %s in state %s", instanceID, e.definition.InitialState)

	return instance, nil
}

// SendEvent sends an event to a state machine instance.
func (e *Engine) SendEvent(ctx context.Context, instanceID string, event *Event) (*TransitionResult, error) {
	e.mu.Lock()
	instance, ok := e.instances[instanceID]
	e.mu.Unlock()

	if !ok {
		// Try to load from persistence
		if e.config.EnablePersistence && e.config.PersistenceStore != nil {
			loaded, err := e.config.PersistenceStore.Load(ctx, instanceID)
			if err != nil {
				return nil, fmt.Errorf("state machine instance not found: %s", instanceID)
			}
			instance = loaded
			e.mu.Lock()
			e.instances[instanceID] = instance
			e.mu.Unlock()
		} else {
			return nil, fmt.Errorf("state machine instance not found: %s", instanceID)
		}
	}

	// Check if instance is active
	if instance.Status != InstanceStatusActive {
		return nil, fmt.Errorf("state machine instance %s is not active (status: %s)", instanceID, instance.Status)
	}

	// Set request ID if available
	if event.RequestID == "" {
		event.RequestID = core.GetRequestID(ctx)
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Find applicable transitions
	transitions := e.findTransitions(instance.Context.CurrentState, event.Name)
	if len(transitions) == 0 {
		e.logger.Warnf("No transition found for state %s and event %s", instance.Context.CurrentState, event.Name)
		return &TransitionResult{
			Success:   false,
			FromState: instance.Context.CurrentState,
			ToState:   instance.Context.CurrentState,
			Event:     event.Name,
			Error:     fmt.Errorf("no transition found for event %s in state %s", event.Name, instance.Context.CurrentState),
			Timestamp: time.Now(),
		}, nil
	}

	// Evaluate guards and find the first valid transition
	var selectedTransition *Transition
	for _, transition := range transitions {
		if transition.Guard != nil {
			canTransition, err := transition.Guard(instance.Context, event)
			if err != nil {
				e.logger.Errorf("Error evaluating guard for transition %s: %v", transition.ID, err)
				continue
			}
			if !canTransition {
				continue
			}
		}
		selectedTransition = transition
		break
	}

	if selectedTransition == nil {
		e.logger.Warnf("No valid transition found (guards failed) for state %s and event %s", instance.Context.CurrentState, event.Name)
		return &TransitionResult{
			Success:   false,
			FromState: instance.Context.CurrentState,
			ToState:   instance.Context.CurrentState,
			Event:     event.Name,
			Error:     fmt.Errorf("transition guards failed for event %s in state %s", event.Name, instance.Context.CurrentState),
			Timestamp: time.Now(),
		}, nil
	}

	// Execute the transition
	result := e.executeTransition(ctx, instance, selectedTransition, event)

	// Persist if enabled
	if e.config.EnablePersistence && e.config.PersistenceStore != nil {
		instance.UpdatedAt = time.Now()
		if err := e.config.PersistenceStore.Save(ctx, instance); err != nil {
			e.logger.Errorf("Failed to persist state machine instance %s: %v", instanceID, err)
		}
	}

	// Publish transition event
	if e.config.EnableEventBus && e.eventBus != nil {
		e.publishEvent("transition.completed", map[string]interface{}{
			"machineId":    instanceID,
			"fromState":    result.FromState,
			"toState":      result.ToState,
			"event":        result.Event,
			"success":      result.Success,
			"transitionId": result.TransitionID,
			"timestamp":    result.Timestamp,
		})
	}

	return result, nil
}

// executeTransition performs a state transition.
func (e *Engine) executeTransition(ctx context.Context, instance *StateMachineInstance, transition *Transition, event *Event) *TransitionResult {
	fromState := instance.Context.CurrentState
	toState := transition.To

	e.logger.Infof("Transitioning instance %s from %s to %s via event %s", instance.ID, fromState, toState, event.Name)

	// Execute OnExit for current state
	currentState := e.definition.States[fromState]
	if currentState != nil && currentState.OnExit != nil {
		if err := currentState.OnExit(instance.Context); err != nil {
			e.logger.Errorf("Error executing OnExit for state %s: %v", fromState, err)
			return &TransitionResult{
				Success:      false,
				FromState:    fromState,
				ToState:      fromState,
				Event:        event.Name,
				Error:        fmt.Errorf("OnExit failed: %w", err),
				TransitionID: transition.ID,
				Timestamp:    time.Now(),
			}
		}
	}

	// Execute transition action
	if transition.Action != nil {
		if err := transition.Action(instance.Context, event); err != nil {
			e.logger.Errorf("Error executing transition action for %s: %v", transition.ID, err)
			return &TransitionResult{
				Success:      false,
				FromState:    fromState,
				ToState:      fromState,
				Event:        event.Name,
				Error:        fmt.Errorf("transition action failed: %w", err),
				TransitionID: transition.ID,
				Timestamp:    time.Now(),
			}
		}
	}

	// Update state
	previousState := instance.Context.CurrentState
	instance.Context.PreviousState = &previousState
	instance.Context.CurrentState = toState
	instance.Context.LastTransitionTime = time.Now()

	// Record history
	if e.config.EnableHistory {
		historyEntry := &HistoryEntry{
			FromState:    fromState,
			ToState:      toState,
			Event:        event.Name,
			Timestamp:    time.Now(),
			TransitionID: transition.ID,
			Data:         copyMap(instance.Context.Data),
		}
		instance.Context.History = append(instance.Context.History, historyEntry)

		// Trim history if needed
		if e.config.MaxHistorySize > 0 && len(instance.Context.History) > e.config.MaxHistorySize {
			instance.Context.History = instance.Context.History[len(instance.Context.History)-e.config.MaxHistorySize:]
		}
	}

	// Execute OnEnter for new state
	newState := e.definition.States[toState]
	if newState != nil && newState.OnEnter != nil {
		if err := newState.OnEnter(instance.Context); err != nil {
			e.logger.Errorf("Error executing OnEnter for state %s: %v", toState, err)
			// Note: We don't rollback the state change, but we record the error
			return &TransitionResult{
				Success:      false,
				FromState:    fromState,
				ToState:      toState,
				Event:        event.Name,
				Error:        fmt.Errorf("OnEnter failed: %w", err),
				TransitionID: transition.ID,
				Timestamp:    time.Now(),
			}
		}
	}

	// Check if reached final state
	if newState != nil && newState.IsFinal {
		instance.Status = InstanceStatusCompleted
		now := time.Now()
		instance.CompletedAt = &now
		e.logger.Infof("State machine instance %s completed in final state %s", instance.ID, toState)
	}

	return &TransitionResult{
		Success:      true,
		FromState:    fromState,
		ToState:      toState,
		Event:        event.Name,
		TransitionID: transition.ID,
		Timestamp:    time.Now(),
	}
}

// findTransitions finds all transitions matching the current state and event.
func (e *Engine) findTransitions(currentState StateType, event TransitionEvent) []*Transition {
	var matches []*Transition

	for _, transition := range e.definition.Transitions {
		if transition.From == currentState && transition.Event == event {
			matches = append(matches, transition)
		}
	}

	// Sort by priority (higher priority first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Priority > matches[j].Priority
	})

	return matches
}

// GetInstance retrieves a state machine instance.
func (e *Engine) GetInstance(ctx context.Context, instanceID string) (*StateMachineInstance, error) {
	e.mu.RLock()
	instance, ok := e.instances[instanceID]
	e.mu.RUnlock()

	if !ok {
		// Try to load from persistence
		if e.config.EnablePersistence && e.config.PersistenceStore != nil {
			loaded, err := e.config.PersistenceStore.Load(ctx, instanceID)
			if err != nil {
				return nil, fmt.Errorf("state machine instance not found: %s", instanceID)
			}
			return loaded, nil
		}
		return nil, fmt.Errorf("state machine instance not found: %s", instanceID)
	}

	return instance, nil
}

// GetCurrentState returns the current state of an instance.
func (e *Engine) GetCurrentState(ctx context.Context, instanceID string) (StateType, error) {
	instance, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return "", err
	}
	return instance.Context.CurrentState, nil
}

// CanTransition checks if a transition is possible for a given event.
func (e *Engine) CanTransition(ctx context.Context, instanceID string, event *Event) (bool, error) {
	instance, err := e.GetInstance(ctx, instanceID)
	if err != nil {
		return false, err
	}

	if instance.Status != InstanceStatusActive {
		return false, nil
	}

	transitions := e.findTransitions(instance.Context.CurrentState, event.Name)
	if len(transitions) == 0 {
		return false, nil
	}

	// Check if any transition's guard passes
	for _, transition := range transitions {
		if transition.Guard != nil {
			canTransition, err := transition.Guard(instance.Context, event)
			if err != nil {
				return false, err
			}
			if canTransition {
				return true, nil
			}
		} else {
			// No guard means transition is always possible
			return true, nil
		}
	}

	return false, nil
}

// Stop stops the state machine engine.
func (e *Engine) Stop() error {
	e.cancel()
	e.logger.Infof("Stopped state machine engine for definition %s", e.definition.ID)
	return nil
}

// registerEventBusConsumers sets up EventBus consumers for the state machine.
func (e *Engine) registerEventBusConsumers() {
	if e.eventBus == nil {
		return
	}

	prefix := e.config.EventBusPrefix
	if prefix == "" {
		prefix = "statemachine"
	}

	// Consumer for sending events to instances
	address := fmt.Sprintf("%s.%s.event", prefix, e.definition.ID)
	e.eventBus.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			InstanceID string                 `json:"instanceId"`
			Event      string                 `json:"event"`
			Data       map[string]interface{} `json:"data"`
		}

		body, ok := msg.Body().([]byte)
		if !ok {
			return fmt.Errorf("invalid message body type")
		}

		if err := core.JSONDecode(body, &req); err != nil {
			return err
		}

		event := &Event{
			Name:      TransitionEvent(req.Event),
			Data:      req.Data,
			Timestamp: time.Now(),
			Source:    "eventbus",
			RequestID: core.GetRequestID(ctx.Context()),
		}

		result, err := e.SendEvent(ctx.Context(), req.InstanceID, event)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{
			"success":      result.Success,
			"fromState":    result.FromState,
			"toState":      result.ToState,
			"transitionId": result.TransitionID,
		})
	})

	e.logger.Infof("Registered EventBus consumer at address: %s", address)
}

// publishEvent publishes an event to the EventBus.
func (e *Engine) publishEvent(eventType string, data map[string]interface{}) {
	if e.eventBus == nil {
		return
	}

	prefix := e.config.EventBusPrefix
	if prefix == "" {
		prefix = "statemachine"
	}

	address := fmt.Sprintf("%s.%s.%s", prefix, e.definition.ID, eventType)
	if err := e.eventBus.Publish(address, data); err != nil {
		e.logger.Errorf("Failed to publish event to %s: %v", address, err)
	}
}

// validateDefinition validates a state machine definition.
func validateDefinition(def *StateMachineDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("state machine ID is required")
	}

	if len(def.States) == 0 {
		return fmt.Errorf("state machine must have at least one state")
	}

	if _, ok := def.States[def.InitialState]; !ok {
		return fmt.Errorf("initial state %s not found in states", def.InitialState)
	}

	// Validate transitions reference valid states
	for _, transition := range def.Transitions {
		if transition.ID == "" {
			return fmt.Errorf("transition ID is required")
		}
		if _, ok := def.States[transition.From]; !ok {
			return fmt.Errorf("transition %s references unknown from state: %s", transition.ID, transition.From)
		}
		if _, ok := def.States[transition.To]; !ok {
			return fmt.Errorf("transition %s references unknown to state: %s", transition.ID, transition.To)
		}
	}

	return nil
}

// DefaultConfig returns default configuration.
func DefaultConfig() *StateMachineConfig {
	return &StateMachineConfig{
		EnableHistory:     true,
		MaxHistorySize:    100,
		EnablePersistence: false,
		EnableEventBus:    true,
		EventBusPrefix:    "statemachine",
		DefaultTimeout:    30 * time.Second,
	}
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
