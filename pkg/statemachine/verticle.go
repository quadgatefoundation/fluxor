package statemachine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// StateMachineVerticle is a Verticle that manages state machines.
type StateMachineVerticle struct {
	eventBus   core.EventBus
	vertx      core.Vertx
	machines   map[string]StateMachine
	definitions map[string]*Definition
	mu         sync.RWMutex
	logger     core.Logger
	httpAddr   string
	server     *web.FastHTTPServer
	persistence PersistenceAdapter
}

// StateMachineVerticleConfig configures the state machine verticle.
type StateMachineVerticleConfig struct {
	HTTPAddr    string             // Optional HTTP API address
	Persistence PersistenceAdapter // Optional persistence adapter
}

// NewStateMachineVerticle creates a new state machine verticle.
func NewStateMachineVerticle(config *StateMachineVerticleConfig) *StateMachineVerticle {
	verticle := &StateMachineVerticle{
		machines:    make(map[string]StateMachine),
		definitions: make(map[string]*Definition),
		logger:      core.NewDefaultLogger(),
	}
	
	if config != nil {
		verticle.httpAddr = config.HTTPAddr
		verticle.persistence = config.Persistence
	}
	
	return verticle
}

// Start starts the verticle.
func (v *StateMachineVerticle) Start(ctx core.FluxorContext) error {
	v.eventBus = ctx.EventBus()
	v.vertx = ctx.Vertx()
	
	v.logger.Info("Starting StateMachineVerticle")
	
	// Register EventBus consumers
	v.registerEventBusConsumers()
	
	// Start HTTP server if configured
	if v.httpAddr != "" {
		if err := v.startHTTPServer(); err != nil {
			return fmt.Errorf("failed to start HTTP server: %w", err)
		}
	}
	
	return nil
}

// Stop stops the verticle.
func (v *StateMachineVerticle) Stop(ctx core.FluxorContext) error {
	v.logger.Info("Stopping StateMachineVerticle")
	
	// Stop all state machines
	v.mu.Lock()
	for id, machine := range v.machines {
		if err := machine.Stop(ctx.Context()); err != nil {
			v.logger.Errorf("Failed to stop state machine %s: %v", id, err)
		}
	}
	v.mu.Unlock()
	
	// Stop HTTP server
	if v.server != nil {
		return v.server.Stop()
	}
	
	return nil
}

// RegisterDefinition registers a state machine definition.
func (v *StateMachineVerticle) RegisterDefinition(def *Definition) error {
	if err := validateDefinition(def); err != nil {
		return fmt.Errorf("invalid definition: %w", err)
	}
	
	v.mu.Lock()
	v.definitions[def.ID] = def
	v.mu.Unlock()
	
	v.logger.Infof("Registered state machine definition: %s", def.ID)
	return nil
}

// CreateMachine creates a new state machine instance.
func (v *StateMachineVerticle) CreateMachine(ctx context.Context, definitionID string, machineID string, initialContext map[string]interface{}) (StateMachine, error) {
	v.mu.RLock()
	def, ok := v.definitions[definitionID]
	v.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("definition not found: %s", definitionID)
	}
	
	opts := []Option{
		WithEventBus(v.eventBus),
		WithLogger(v.logger),
	}
	
	if machineID != "" {
		opts = append(opts, WithID(machineID))
	}
	
	if initialContext != nil {
		opts = append(opts, WithInitialContext(initialContext))
	}
	
	if v.persistence != nil {
		opts = append(opts, WithPersistence(v.persistence))
	}
	
	machine, err := NewStateMachine(def, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}
	
	// Start the machine
	if err := machine.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start state machine: %w", err)
	}
	
	// Store the machine
	v.mu.Lock()
	v.machines[machine.ID()] = machine
	v.mu.Unlock()
	
	v.logger.Infof("Created state machine: %s (definition: %s)", machine.ID(), definitionID)
	return machine, nil
}

// GetMachine gets a state machine by ID.
func (v *StateMachineVerticle) GetMachine(id string) (StateMachine, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	machine, ok := v.machines[id]
	if !ok {
		return nil, fmt.Errorf("state machine not found: %s", id)
	}
	return machine, nil
}

// registerEventBusConsumers registers EventBus consumers for state machine management.
func (v *StateMachineVerticle) registerEventBusConsumers() {
	// Register definition
	v.eventBus.Consumer("statemachine.register").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var def Definition
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &def); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}
		
		if err := v.RegisterDefinition(&def); err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}
		
		return msg.Reply(map[string]interface{}{"success": true, "id": def.ID})
	})
	
	// Create machine
	v.eventBus.Consumer("statemachine.create").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			DefinitionID   string                 `json:"definitionId"`
			MachineID      string                 `json:"machineId,omitempty"`
			InitialContext map[string]interface{} `json:"initialContext,omitempty"`
		}
		
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &req); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}
		
		machine, err := v.CreateMachine(ctx.Context(), req.DefinitionID, req.MachineID, req.InitialContext)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}
		
		return msg.Reply(map[string]interface{}{
			"success":   true,
			"machineId": machine.ID(),
			"state":     machine.CurrentState(),
		})
	})
	
	// Get machine status
	v.eventBus.Consumer("statemachine.status").Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var req struct {
			MachineID string `json:"machineId"`
		}
		
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &req); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}
		
		machine, err := v.GetMachine(req.MachineID)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}
		
		return msg.Reply(map[string]interface{}{
			"success":      true,
			"machineId":    machine.ID(),
			"currentState": machine.CurrentState(),
			"history":      machine.GetHistory(),
		})
	})
	
	v.logger.Info("EventBus consumers registered for StateMachineVerticle")
}

// startHTTPServer starts the HTTP API server.
func (v *StateMachineVerticle) startHTTPServer() error {
	config := web.DefaultFastHTTPServerConfig(v.httpAddr)
	v.server = web.NewFastHTTPServer(v.vertx, config)
	router := v.server.FastRouter()
	
	// Health check
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status": "UP",
		})
	})
	
	// List definitions
	router.GETFast("/definitions", func(c *web.FastRequestContext) error {
		v.mu.RLock()
		defs := make([]*Definition, 0, len(v.definitions))
		for _, def := range v.definitions {
			defs = append(defs, def)
		}
		v.mu.RUnlock()
		
		return c.JSON(200, defs)
	})
	
	// Register definition
	router.POSTFast("/definitions", func(c *web.FastRequestContext) error {
		var def Definition
		if err := json.Unmarshal(c.RequestCtx.PostBody(), &def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid request body"})
		}
		
		if err := v.RegisterDefinition(&def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		
		return c.JSON(201, map[string]interface{}{"id": def.ID})
	})
	
	// Create machine
	router.POSTFast("/machines", func(c *web.FastRequestContext) error {
		var req struct {
			DefinitionID   string                 `json:"definitionId"`
			MachineID      string                 `json:"machineId,omitempty"`
			InitialContext map[string]interface{} `json:"initialContext,omitempty"`
		}
		
		if err := json.Unmarshal(c.RequestCtx.PostBody(), &req); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid request body"})
		}
		
		machine, err := v.CreateMachine(c.Context(), req.DefinitionID, req.MachineID, req.InitialContext)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		
		return c.JSON(201, map[string]interface{}{
			"machineId": machine.ID(),
			"state":     machine.CurrentState(),
		})
	})
	
	// List machines
	router.GETFast("/machines", func(c *web.FastRequestContext) error {
		v.mu.RLock()
		machineList := make([]map[string]interface{}, 0, len(v.machines))
		for id, machine := range v.machines {
			machineList = append(machineList, map[string]interface{}{
				"id":           id,
				"currentState": machine.CurrentState(),
				"definition":   machine.GetDefinition().ID,
			})
		}
		v.mu.RUnlock()
		
		return c.JSON(200, machineList)
	})
	
	// Get machine status
	router.GETFast("/machines/:id", func(c *web.FastRequestContext) error {
		machineID := c.Params["id"]
		
		machine, err := v.GetMachine(machineID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": "machine not found"})
		}
		
		return c.JSON(200, map[string]interface{}{
			"id":           machine.ID(),
			"currentState": machine.CurrentState(),
			"definition":   machine.GetDefinition().ID,
			"history":      machine.GetHistory(),
		})
	})
	
	// Send event to machine
	router.POSTFast("/machines/:id/events", func(c *web.FastRequestContext) error {
		machineID := c.Params["id"]
		
		var event Event
		if err := json.Unmarshal(c.RequestCtx.PostBody(), &event); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid request body"})
		}
		
		machine, err := v.GetMachine(machineID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": "machine not found"})
		}
		
		if err := machine.Send(c.Context(), event); err != nil {
			if smErr, ok := err.(*StateMachineError); ok {
				return c.JSON(400, map[string]interface{}{
					"error": smErr.Message,
					"code":  smErr.Code,
				})
			}
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}
		
		return c.JSON(200, map[string]interface{}{
			"success":      true,
			"currentState": machine.CurrentState(),
		})
	})
	
	// Reset machine
	router.POSTFast("/machines/:id/reset", func(c *web.FastRequestContext) error {
		machineID := c.Params["id"]
		
		machine, err := v.GetMachine(machineID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": "machine not found"})
		}
		
		if err := machine.Reset(c.Context()); err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}
		
		return c.JSON(200, map[string]interface{}{
			"success":      true,
			"currentState": machine.CurrentState(),
		})
	})
	
	// Start server
	go v.server.Start()
	
	v.logger.Infof("HTTP API server started on %s", v.httpAddr)
	return nil
}
