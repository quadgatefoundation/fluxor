package statemachine

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/google/uuid"
)

// Engine implements StateMachineEngine using EventBus.
type Engine struct {
	eventBus    core.EventBus
	machines    map[string]*StateMachineDefinition
	instances   map[string]*ExecutionContext
	guards      map[string]GuardFunc
	actions     map[string]ActionFunc
	handlers    map[string]StateHandler
	listeners   []StateChangeListener
	persistence PersistenceProvider
	mu          sync.RWMutex
	logger      core.Logger
}

// NewEngine creates a new state machine engine.
func NewEngine(eventBus core.EventBus) *Engine {
	return &Engine{
		eventBus:  eventBus,
		machines:  make(map[string]*StateMachineDefinition),
		instances: make(map[string]*ExecutionContext),
		guards:    make(map[string]GuardFunc),
		actions:   make(map[string]ActionFunc),
		handlers:  make(map[string]StateHandler),
		listeners: make([]StateChangeListener, 0),
		logger:    core.NewDefaultLogger(),
	}
}

// RegisterMachine registers a state machine definition.
func (e *Engine) RegisterMachine(def *StateMachineDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("machine ID is required")
	}
	if def.InitialState == "" {
		return fmt.Errorf("initial state is required")
	}
	if len(def.States) == 0 {
		return fmt.Errorf("machine must have at least one state")
	}

	// Validate states and transitions
	stateIDs := make(map[string]bool)
	for _, state := range def.States {
		if state.ID == "" {
			return fmt.Errorf("state ID is required")
		}
		stateIDs[state.ID] = true
	}

	// Check initial state exists
	if !stateIDs[def.InitialState] {
		return fmt.Errorf("initial state %s not found in states", def.InitialState)
	}

	// Validate transitions
	for _, state := range def.States {
		for _, trans := range state.Transitions {
			if trans.Event == "" {
				return fmt.Errorf("transition event is required in state %s", state.ID)
			}
			if trans.Target == "" {
				return fmt.Errorf("transition target is required for event %s in state %s", trans.Event, state.ID)
			}
			if !stateIDs[trans.Target] {
				return fmt.Errorf("transition target %s not found in state %s", trans.Target, state.ID)
			}
		}
	}

	e.mu.Lock()
	e.machines[def.ID] = def
	e.mu.Unlock()

	// Register EventBus consumers for this machine
	e.registerMachineConsumers(def)

	return nil
}

// registerMachineConsumers sets up EventBus consumers for state machine events.
func (e *Engine) registerMachineConsumers(def *StateMachineDefinition) {
	// Consumer for creating new instances
	createAddress := fmt.Sprintf("statemachine.%s.create", def.ID)
	e.eventBus.Consumer(createAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			InitialData map[string]interface{} `json:"initialData"`
		}
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &req); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}

		instanceID, err := e.CreateInstance(ctx.Context(), def.ID, req.InitialData)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{
			"instanceId":   instanceID,
			"currentState": def.InitialState,
		})
	})

	// Consumer for sending events to instances
	eventAddress := fmt.Sprintf("statemachine.%s.event", def.ID)
	e.eventBus.Consumer(eventAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			InstanceID string                 `json:"instanceId"`
			Event      string                 `json:"event"`
			Data       map[string]interface{} `json:"data"`
		}
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &req); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}

		event := &Event{
			Name:      req.Event,
			Data:      req.Data,
			Timestamp: time.Now(),
		}

		if err := e.SendEvent(ctx.Context(), req.InstanceID, event); err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		// Get updated instance
		instance, err := e.GetInstance(req.InstanceID)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{
			"instanceId":   req.InstanceID,
			"currentState": instance.CurrentState,
			"status":       instance.Status,
		})
	})

	// Consumer for querying instance state
	queryAddress := fmt.Sprintf("statemachine.%s.query", def.ID)
	e.eventBus.Consumer(queryAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			InstanceID string `json:"instanceId"`
		}
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &req); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}

		instance, err := e.GetInstance(req.InstanceID)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(instance)
	})
}

// CreateInstance creates a new state machine instance.
func (e *Engine) CreateInstance(ctx context.Context, machineID string, initialData map[string]interface{}) (string, error) {
	e.mu.RLock()
	def, ok := e.machines[machineID]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("machine not found: %s", machineID)
	}

	instanceID := uuid.New().String()
	now := time.Now()

	execCtx := &ExecutionContext{
		MachineID:     machineID,
		InstanceID:    instanceID,
		CurrentState:  def.InitialState,
		PreviousState: "",
		StartTime:     now,
		UpdateTime:    now,
		Data:          initialData,
		Variables:     make(map[string]interface{}),
		History:       make([]StateTransition, 0),
		Status:        ExecutionStatusRunning,
		ActiveStates:  []string{def.InitialState},
	}

	if execCtx.Data == nil {
		execCtx.Data = make(map[string]interface{})
	}

	e.mu.Lock()
	e.instances[instanceID] = execCtx
	e.mu.Unlock()

	// Execute onEnter actions for initial state
	state := e.findState(def, def.InitialState)
	if state != nil {
		event := &Event{
			Name:      "__init__",
			Data:      initialData,
			Timestamp: now,
		}
		if err := e.executeOnEnterActions(ctx, state, event, execCtx); err != nil {
			e.logger.Errorf("failed to execute onEnter for initial state: %v", err)
		}
	}

	// Notify listeners
	e.notifyStateChange(ctx, instanceID, "", def.InitialState, &Event{Name: "__init__", Timestamp: now})

	// Persist if provider is configured
	if e.persistence != nil {
		if err := e.persistence.Save(instanceID, execCtx); err != nil {
			e.logger.Errorf("failed to persist instance: %v", err)
		}
	}

	return instanceID, nil
}

// SendEvent sends an event to a state machine instance.
func (e *Engine) SendEvent(ctx context.Context, instanceID string, event *Event) error {
	e.mu.Lock()
	execCtx, ok := e.instances[instanceID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	// Check if instance is in a terminal state
	if execCtx.Status != ExecutionStatusRunning {
		e.mu.Unlock()
		return fmt.Errorf("instance %s is not running (status: %s)", instanceID, execCtx.Status)
	}

	def, ok := e.machines[execCtx.MachineID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("machine not found: %s", execCtx.MachineID)
	}

	// Copy instance to avoid race conditions during processing
	execCtxCopy := *execCtx
	e.mu.Unlock()

	// Find current state
	currentState := e.findState(def, execCtxCopy.CurrentState)
	if currentState == nil {
		return fmt.Errorf("current state %s not found", execCtxCopy.CurrentState)
	}

	// Find matching transition
	transition := e.findTransition(ctx, currentState, event, &execCtxCopy)
	if transition == nil {
		e.logger.Infof("no transition found for event %s in state %s", event.Name, currentState.ID)
		return nil // Not an error - just no transition
	}

	// Execute state transition
	if err := e.executeTransition(ctx, def, currentState, transition, event, &execCtxCopy); err != nil {
		return fmt.Errorf("failed to execute transition: %w", err)
	}

	// Update instance
	e.mu.Lock()
	e.instances[instanceID] = &execCtxCopy
	e.mu.Unlock()

	// Persist if provider is configured
	if e.persistence != nil {
		if err := e.persistence.Save(instanceID, &execCtxCopy); err != nil {
			e.logger.Errorf("failed to persist instance: %v", err)
		}
	}

	return nil
}

// findTransition finds a matching transition for the event.
func (e *Engine) findTransition(ctx context.Context, state *StateDefinition, event *Event, execCtx *ExecutionContext) *TransitionDefinition {
	// Sort transitions by priority (higher first)
	transitions := make([]TransitionDefinition, len(state.Transitions))
	copy(transitions, state.Transitions)
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].Priority > transitions[j].Priority
	})

	// Find first matching transition
	for _, trans := range transitions {
		if trans.Event == event.Name {
			// Check guard if present
			if trans.Guard != "" {
				guard, ok := e.guards[trans.Guard]
				if !ok {
					e.logger.Warnf("guard %s not found, transition denied", trans.Guard)
					continue
				}
				if !guard(ctx, event, execCtx) {
					continue
				}
			}
			return &trans
		}
	}

	return nil
}

// executeTransition executes a state transition.
func (e *Engine) executeTransition(ctx context.Context, def *StateMachineDefinition, fromState *StateDefinition, transition *TransitionDefinition, event *Event, execCtx *ExecutionContext) error {
	targetState := e.findState(def, transition.Target)
	if targetState == nil {
		return fmt.Errorf("target state %s not found", transition.Target)
	}

	// Execute onExit actions for current state
	if err := e.executeOnExitActions(ctx, fromState, event, execCtx); err != nil {
		return fmt.Errorf("onExit failed: %w", err)
	}

	// Execute transition actions
	for _, actionDef := range transition.Actions {
		if err := e.executeAction(ctx, &actionDef, event, execCtx); err != nil {
			return fmt.Errorf("transition action %s failed: %w", actionDef.Name, err)
		}
	}

	// Update state
	previousState := execCtx.CurrentState
	execCtx.PreviousState = previousState
	execCtx.CurrentState = transition.Target
	execCtx.UpdateTime = time.Now()
	execCtx.ActiveStates = []string{transition.Target}

	// Record transition in history
	execCtx.History = append(execCtx.History, StateTransition{
		FromState: previousState,
		ToState:   transition.Target,
		Event:     event.Name,
		Timestamp: time.Now(),
		Data:      event.Data,
	})

	// Execute onEnter actions for target state
	if err := e.executeOnEnterActions(ctx, targetState, event, execCtx); err != nil {
		return fmt.Errorf("onEnter failed: %w", err)
	}

	// Check if target is a final state
	if targetState.Type == StateTypeFinal {
		execCtx.Status = ExecutionStatusCompleted
	}

	// Notify listeners
	e.notifyStateChange(ctx, execCtx.InstanceID, previousState, transition.Target, event)

	e.logger.Infof("state transition: %s -> %s (event: %s, instance: %s)",
		previousState, transition.Target, event.Name, execCtx.InstanceID)

	return nil
}

// executeOnEnterActions executes onEnter actions for a state.
func (e *Engine) executeOnEnterActions(ctx context.Context, state *StateDefinition, event *Event, execCtx *ExecutionContext) error {
	// Execute registered handler
	if handler, ok := e.handlers[state.ID]; ok {
		if err := handler.OnEnter(ctx, event, execCtx); err != nil {
			return err
		}
	}

	// Execute configured actions
	for _, actionDef := range state.OnEnter {
		if err := e.executeAction(ctx, &actionDef, event, execCtx); err != nil {
			return err
		}
	}

	return nil
}

// executeOnExitActions executes onExit actions for a state.
func (e *Engine) executeOnExitActions(ctx context.Context, state *StateDefinition, event *Event, execCtx *ExecutionContext) error {
	// Execute configured actions
	for _, actionDef := range state.OnExit {
		if err := e.executeAction(ctx, &actionDef, event, execCtx); err != nil {
			return err
		}
	}

	// Execute registered handler
	if handler, ok := e.handlers[state.ID]; ok {
		if err := handler.OnExit(ctx, event, execCtx); err != nil {
			return err
		}
	}

	return nil
}

// executeAction executes a single action.
func (e *Engine) executeAction(ctx context.Context, actionDef *ActionDefinition, event *Event, execCtx *ExecutionContext) error {
	switch actionDef.Type {
	case "function":
		action, ok := e.actions[actionDef.Name]
		if !ok {
			return fmt.Errorf("action %s not found", actionDef.Name)
		}
		return action(ctx, event, execCtx)

	case "eventbus":
		address, ok := actionDef.Config["address"].(string)
		if !ok {
			return fmt.Errorf("eventbus action requires 'address' in config")
		}
		data := map[string]interface{}{
			"instanceId": execCtx.InstanceID,
			"event":      event.Name,
			"data":       event.Data,
		}
		return e.eventBus.Publish(address, data)

	case "set":
		// Set variables
		if values, ok := actionDef.Config["values"].(map[string]interface{}); ok {
			for k, v := range values {
				execCtx.Variables[k] = v
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown action type: %s", actionDef.Type)
	}
}

// findState finds a state by ID.
func (e *Engine) findState(def *StateMachineDefinition, stateID string) *StateDefinition {
	for i := range def.States {
		if def.States[i].ID == stateID {
			return &def.States[i]
		}
	}
	return nil
}

// GetInstance returns the execution context for an instance.
func (e *Engine) GetInstance(instanceID string) (*ExecutionContext, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	execCtx, ok := e.instances[instanceID]
	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	return execCtx, nil
}

// ListInstances lists all instances for a machine.
func (e *Engine) ListInstances(machineID string) ([]*ExecutionContext, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*ExecutionContext, 0)
	for _, instance := range e.instances {
		if instance.MachineID == machineID {
			result = append(result, instance)
		}
	}

	return result, nil
}

// SaveInstance persists instance state.
func (e *Engine) SaveInstance(instanceID string) error {
	if e.persistence == nil {
		return fmt.Errorf("persistence provider not configured")
	}

	e.mu.RLock()
	execCtx, ok := e.instances[instanceID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	return e.persistence.Save(instanceID, execCtx)
}

// RestoreInstance restores instance state from persistence.
func (e *Engine) RestoreInstance(instanceID string) error {
	if e.persistence == nil {
		return fmt.Errorf("persistence provider not configured")
	}

	execCtx, err := e.persistence.Load(instanceID)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.instances[instanceID] = execCtx
	e.mu.Unlock()

	return nil
}

// RegisterGuard registers a guard function.
func (e *Engine) RegisterGuard(name string, guard GuardFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.guards[name] = guard
}

// RegisterAction registers an action function.
func (e *Engine) RegisterAction(name string, action ActionFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.actions[name] = action
}

// RegisterStateHandler registers a state handler.
func (e *Engine) RegisterStateHandler(stateID string, handler StateHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[stateID] = handler
}

// AddStateChangeListener adds a state change listener.
func (e *Engine) AddStateChangeListener(listener StateChangeListener) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners = append(e.listeners, listener)
}

// SetPersistence sets the persistence provider.
func (e *Engine) SetPersistence(provider PersistenceProvider) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.persistence = provider
}

// notifyStateChange notifies all listeners of a state change.
func (e *Engine) notifyStateChange(ctx context.Context, instanceID, fromState, toState string, event *Event) {
	e.mu.RLock()
	listeners := make([]StateChangeListener, len(e.listeners))
	copy(listeners, e.listeners)
	e.mu.RUnlock()

	for _, listener := range listeners {
		// Execute listener in goroutine to avoid blocking
		go func(l StateChangeListener) {
			defer func() {
				if r := recover(); r != nil {
					e.logger.Errorf("state change listener panicked: %v", r)
				}
			}()
			l(ctx, instanceID, fromState, toState, event)
		}(listener)
	}
}
