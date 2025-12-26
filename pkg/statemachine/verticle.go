package statemachine

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// Verticle is a deployable state machine unit for Fluxor.
type Verticle struct {
	engine     *Engine
	httpAddr   string
	httpServer *web.FastHTTPServer
}

// VerticleConfig configures the state machine verticle.
type VerticleConfig struct {
	HTTPAddr    string              // Optional HTTP API address (e.g., ":8082")
	Persistence PersistenceProvider // Optional persistence provider
}

// NewVerticle creates a new state machine verticle.
func NewVerticle(config *VerticleConfig) *Verticle {
	v := &Verticle{}

	if config != nil {
		v.httpAddr = config.HTTPAddr
	}

	return v
}

// Start initializes the state machine verticle.
func (v *Verticle) Start(ctx core.FluxorContext) error {
	// Create engine with EventBus
	v.engine = NewEngine(ctx.EventBus())

	// Set persistence if configured
	if config, ok := ctx.Config()["persistence"].(PersistenceProvider); ok {
		v.engine.SetPersistence(config)
	}

	// Start HTTP API if configured
	if v.httpAddr != "" {
		if err := v.startHTTPAPI(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop shuts down the state machine verticle.
func (v *Verticle) Stop(ctx core.FluxorContext) error {
	if v.httpServer != nil {
		return v.httpServer.Stop()
	}
	return nil
}

// Engine returns the state machine engine.
func (v *Verticle) Engine() StateMachineEngine {
	return v.engine
}

// RegisterGuard registers a guard function.
func (v *Verticle) RegisterGuard(name string, guard GuardFunc) {
	v.engine.RegisterGuard(name, guard)
}

// RegisterAction registers an action function.
func (v *Verticle) RegisterAction(name string, action ActionFunc) {
	v.engine.RegisterAction(name, action)
}

// RegisterStateHandler registers a state handler.
func (v *Verticle) RegisterStateHandler(stateID string, handler StateHandler) {
	v.engine.RegisterStateHandler(stateID, handler)
}

// AddStateChangeListener adds a state change listener.
func (v *Verticle) AddStateChangeListener(listener StateChangeListener) {
	v.engine.AddStateChangeListener(listener)
}

// startHTTPAPI starts the HTTP API server for state machine management.
func (v *Verticle) startHTTPAPI(ctx core.FluxorContext) error {
	config := web.DefaultFastHTTPServerConfig(v.httpAddr)
	v.httpServer = web.NewFastHTTPServer(ctx.Vertx(), config)
	router := v.httpServer.FastRouter()

	// Health check
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status": "UP",
			"service": "statemachine",
		})
	})

	// Register machine
	router.POSTFast("/machines", func(c *web.FastRequestContext) error {
		var def StateMachineDefinition
		if err := c.BindJSON(&def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		if err := v.engine.RegisterMachine(&def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(201, map[string]interface{}{
			"id":   def.ID,
			"name": def.Name,
			"message": "Machine registered successfully",
		})
	})

	// List machines
	router.GETFast("/machines", func(c *web.FastRequestContext) error {
		v.engine.mu.RLock()
		machines := make([]*StateMachineDefinition, 0, len(v.engine.machines))
		for _, m := range v.engine.machines {
			machines = append(machines, m)
		}
		v.engine.mu.RUnlock()

		return c.JSON(200, map[string]interface{}{
			"machines": machines,
		})
	})

	// Create instance
	router.POSTFast("/machines/:id/instances", func(c *web.FastRequestContext) error {
		machineID := c.Params["id"]

		var req struct {
			InitialData map[string]interface{} `json:"initialData"`
		}
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		instanceID, err := v.engine.CreateInstance(c.Context(), machineID, req.InitialData)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}

		instance, _ := v.engine.GetInstance(instanceID)
		return c.JSON(201, map[string]interface{}{
			"instanceId":   instanceID,
			"currentState": instance.CurrentState,
			"status":       instance.Status,
		})
	})

	// List instances
	router.GETFast("/machines/:id/instances", func(c *web.FastRequestContext) error {
		machineID := c.Params["id"]

		instances, err := v.engine.ListInstances(machineID)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(200, map[string]interface{}{
			"instances": instances,
			"count":     len(instances),
		})
	})

	// Get instance
	router.GETFast("/instances/:id", func(c *web.FastRequestContext) error {
		instanceID := c.Params["id"]

		instance, err := v.engine.GetInstance(instanceID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(200, instance)
	})

	// Send event
	router.POSTFast("/instances/:id/events", func(c *web.FastRequestContext) error {
		instanceID := c.Params["id"]

		var event Event
		if err := c.BindJSON(&event); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		if err := v.engine.SendEvent(c.Context(), instanceID, &event); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}

		instance, _ := v.engine.GetInstance(instanceID)
		return c.JSON(200, map[string]interface{}{
			"instanceId":   instanceID,
			"currentState": instance.CurrentState,
			"status":       instance.Status,
		})
	})

	// Get instance history
	router.GETFast("/instances/:id/history", func(c *web.FastRequestContext) error {
		instanceID := c.Params["id"]

		instance, err := v.engine.GetInstance(instanceID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(200, map[string]interface{}{
			"instanceId": instanceID,
			"history":    instance.History,
		})
	})

	go v.httpServer.Start()
	return nil
}
