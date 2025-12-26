package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// StateMachineVerticle is a Verticle that manages state machines.
type StateMachineVerticle struct {
	engines map[string]*Engine
	mu      sync.RWMutex
	eventBus core.EventBus
	logger  core.Logger
}

// NewStateMachineVerticle creates a new state machine verticle.
func NewStateMachineVerticle() *StateMachineVerticle {
	return &StateMachineVerticle{
		engines: make(map[string]*Engine),
		logger:  core.NewDefaultLogger(),
	}
}

// Start starts the state machine verticle.
func (v *StateMachineVerticle) Start(ctx core.FluxorContext) error {
	v.eventBus = ctx.EventBus()
	v.logger.Infof("Starting StateMachine Verticle")

	// Register global management consumers
	v.registerManagementConsumers(ctx)

	return nil
}

// Stop stops the state machine verticle.
func (v *StateMachineVerticle) Stop(ctx core.FluxorContext) error {
	v.logger.Infof("Stopping StateMachine Verticle")

	// Stop all engines
	v.mu.RLock()
	engines := make([]*Engine, 0, len(v.engines))
	for _, engine := range v.engines {
		engines = append(engines, engine)
	}
	v.mu.RUnlock()

	for _, engine := range engines {
		if err := engine.Stop(); err != nil {
			v.logger.Errorf("Error stopping engine %s: %v", engine.GetDefinition().ID, err)
		}
	}

	return nil
}

// RegisterStateMachine registers a state machine definition and creates an engine.
func (v *StateMachineVerticle) RegisterStateMachine(definition *StateMachineDefinition, config *StateMachineConfig) error {
	if config == nil {
		config = DefaultConfig()
	}

	engine, err := NewEngine(definition, config, v.eventBus)
	if err != nil {
		return fmt.Errorf("failed to create state machine engine: %w", err)
	}

	v.mu.Lock()
	v.engines[definition.ID] = engine
	v.mu.Unlock()

	v.logger.Infof("Registered state machine: %s", definition.ID)

	return nil
}

// GetEngine retrieves a state machine engine by definition ID.
func (v *StateMachineVerticle) GetEngine(definitionID string) (*Engine, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	engine, ok := v.engines[definitionID]
	return engine, ok
}

// registerManagementConsumers sets up EventBus consumers for management operations.
func (v *StateMachineVerticle) registerManagementConsumers(ctx core.FluxorContext) {
	// Consumer for creating instances
	ctx.EventBus().Consumer("statemachine.create").Handler(func(fluxCtx core.FluxorContext, msg core.Message) error {
		var req struct {
			DefinitionID string                 `json:"definitionId"`
			InitialData  map[string]interface{} `json:"initialData"`
		}

		body, ok := msg.Body().([]byte)
		if !ok {
			return msg.Reply(map[string]interface{}{"error": "invalid message body type"})
		}

		if err := core.JSONDecode(body, &req); err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		engine, ok := v.GetEngine(req.DefinitionID)
		if !ok {
			return msg.Reply(map[string]interface{}{"error": fmt.Sprintf("state machine not found: %s", req.DefinitionID)})
		}

		instance, err := engine.CreateInstance(fluxCtx.Context(), req.InitialData)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{
			"instanceId":  instance.ID,
			"definitionId": instance.DefinitionID,
			"currentState": instance.Context.CurrentState,
			"status":       instance.Status,
		})
	})

	// Consumer for querying instance state
	ctx.EventBus().Consumer("statemachine.query").Handler(func(fluxCtx core.FluxorContext, msg core.Message) error {
		var req struct {
			DefinitionID string `json:"definitionId"`
			InstanceID   string `json:"instanceId"`
		}

		body, ok := msg.Body().([]byte)
		if !ok {
			return msg.Reply(map[string]interface{}{"error": "invalid message body type"})
		}

		if err := core.JSONDecode(body, &req); err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		engine, ok := v.GetEngine(req.DefinitionID)
		if !ok {
			return msg.Reply(map[string]interface{}{"error": fmt.Sprintf("state machine not found: %s", req.DefinitionID)})
		}

		instance, err := engine.GetInstance(fluxCtx.Context(), req.InstanceID)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{
			"instanceId":    instance.ID,
			"definitionId":  instance.DefinitionID,
			"currentState":  instance.Context.CurrentState,
			"previousState": instance.Context.PreviousState,
			"status":        instance.Status,
			"data":          instance.Context.Data,
			"createdAt":     instance.CreatedAt,
			"updatedAt":     instance.UpdatedAt,
		})
	})

	// Consumer for listing definitions
	ctx.EventBus().Consumer("statemachine.list").Handler(func(fluxCtx core.FluxorContext, msg core.Message) error {
		v.mu.RLock()
		definitions := make([]map[string]interface{}, 0, len(v.engines))
		for _, engine := range v.engines {
			def := engine.GetDefinition()
			definitions = append(definitions, map[string]interface{}{
				"id":          def.ID,
				"name":        def.Name,
				"description": def.Description,
				"version":     def.Version,
			})
		}
		v.mu.RUnlock()

		return msg.Reply(map[string]interface{}{
			"definitions": definitions,
			"count":       len(definitions),
		})
	})

	v.logger.Infof("Registered state machine management consumers")
}

// StateMachineClient provides a client interface for interacting with state machines.
type StateMachineClient struct {
	eventBus core.EventBus
	timeout  time.Duration
}

// NewStateMachineClient creates a new state machine client.
func NewStateMachineClient(eventBus core.EventBus) *StateMachineClient {
	return &StateMachineClient{
		eventBus: eventBus,
		timeout:  30 * time.Second,
	}
}

// CreateInstance creates a new state machine instance via EventBus.
func (c *StateMachineClient) CreateInstance(ctx context.Context, definitionID string, initialData map[string]interface{}) (string, error) {
	req := map[string]interface{}{
		"definitionId": definitionID,
		"initialData":  initialData,
	}

	msg, err := c.eventBus.Request("statemachine.create", req, c.timeout)
	if err != nil {
		return "", err
	}

	var resp struct {
		InstanceID string `json:"instanceId"`
		Error      string `json:"error"`
	}

	body, ok := msg.Body().([]byte)
	if !ok {
		return "", fmt.Errorf("invalid message body type")
	}

	if err := core.JSONDecode(body, &resp); err != nil {
		return "", err
	}

	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}

	return resp.InstanceID, nil
}

// SendEvent sends an event to a state machine instance via EventBus.
func (c *StateMachineClient) SendEvent(ctx context.Context, definitionID, instanceID, event string, data map[string]interface{}) (bool, error) {
	req := map[string]interface{}{
		"instanceId": instanceID,
		"event":      event,
		"data":       data,
	}

	address := fmt.Sprintf("statemachine.%s.event", definitionID)
	msg, err := c.eventBus.Request(address, req, c.timeout)
	if err != nil {
		return false, err
	}

	var resp struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}

	body, ok := msg.Body().([]byte)
	if !ok {
		return false, fmt.Errorf("invalid message body type")
	}

	if err := core.JSONDecode(body, &resp); err != nil {
		return false, err
	}

	if resp.Error != "" {
		return false, fmt.Errorf("%s", resp.Error)
	}

	return resp.Success, nil
}

// QueryInstance queries the current state of an instance via EventBus.
func (c *StateMachineClient) QueryInstance(ctx context.Context, definitionID, instanceID string) (map[string]interface{}, error) {
	req := map[string]interface{}{
		"definitionId": definitionID,
		"instanceId":   instanceID,
	}

	msg, err := c.eventBus.Request("statemachine.query", req, c.timeout)
	if err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	body, ok := msg.Body().([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid message body type")
	}

	if err := core.JSONDecode(body, &resp); err != nil {
		return nil, err
	}

	if errorMsg, ok := resp["error"].(string); ok && errorMsg != "" {
		return nil, fmt.Errorf("%s", errorMsg)
	}

	return resp, nil
}

// ListDefinitions lists all registered state machine definitions via EventBus.
func (c *StateMachineClient) ListDefinitions(ctx context.Context) ([]map[string]interface{}, error) {
	msg, err := c.eventBus.Request("statemachine.list", map[string]interface{}{}, c.timeout)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Definitions []map[string]interface{} `json:"definitions"`
		Error       string                   `json:"error"`
	}

	body, ok := msg.Body().([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid message body type")
	}

	if err := core.JSONDecode(body, &resp); err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.Definitions, nil
}
