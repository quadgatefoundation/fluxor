package statemachine

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// MemoryPersistenceAdapter stores state in memory (non-persistent).
type MemoryPersistenceAdapter struct {
	data map[string]*persistedState
	mu   sync.RWMutex
}

type persistedState struct {
	State   string                 `json:"state"`
	Context map[string]interface{} `json:"context"`
}

// NewMemoryPersistenceAdapter creates a new memory persistence adapter.
func NewMemoryPersistenceAdapter() *MemoryPersistenceAdapter {
	return &MemoryPersistenceAdapter{
		data: make(map[string]*persistedState),
	}
}

func (m *MemoryPersistenceAdapter) Save(ctx context.Context, machineID string, state string, context map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.data[machineID] = &persistedState{
		State:   state,
		Context: context,
	}
	return nil
}

func (m *MemoryPersistenceAdapter) Load(ctx context.Context, machineID string) (string, map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, ok := m.data[machineID]
	if !ok {
		return "", nil, fmt.Errorf("state not found for machine: %s", machineID)
	}
	
	return data.State, data.Context, nil
}

func (m *MemoryPersistenceAdapter) Delete(ctx context.Context, machineID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.data, machineID)
	return nil
}

// FilePersistenceAdapter stores state in files.
type FilePersistenceAdapter struct {
	basePath string
	mu       sync.RWMutex
}

// NewFilePersistenceAdapter creates a new file persistence adapter.
func NewFilePersistenceAdapter(basePath string) (*FilePersistenceAdapter, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	
	return &FilePersistenceAdapter{
		basePath: basePath,
	}, nil
}

func (f *FilePersistenceAdapter) Save(ctx context.Context, machineID string, state string, context map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	data := &persistedState{
		State:   state,
		Context: context,
	}
	
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	filePath := filepath.Join(f.basePath, fmt.Sprintf("%s.json", machineID))
	if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	
	return nil
}

func (f *FilePersistenceAdapter) Load(ctx context.Context, machineID string) (string, map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	filePath := filepath.Join(f.basePath, fmt.Sprintf("%s.json", machineID))
	
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("state not found for machine: %s", machineID)
		}
		return "", nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var data persistedState
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}
	
	return data.State, data.Context, nil
}

func (f *FilePersistenceAdapter) Delete(ctx context.Context, machineID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	filePath := filepath.Join(f.basePath, fmt.Sprintf("%s.json", machineID))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file: %w", err)
	}
	
	return nil
}

// EventBusPersistenceAdapter uses EventBus for persistence (delegating to external service).
type EventBusPersistenceAdapter struct {
	eventBus core.EventBus
	address  string
}

// NewEventBusPersistenceAdapter creates a new EventBus persistence adapter.
func NewEventBusPersistenceAdapter(eventBus core.EventBus, address string) *EventBusPersistenceAdapter {
	return &EventBusPersistenceAdapter{
		eventBus: eventBus,
		address:  address,
	}
}

func (e *EventBusPersistenceAdapter) Save(ctx context.Context, machineID string, state string, context map[string]interface{}) error {
	req := map[string]interface{}{
		"operation":  "save",
		"machineId":  machineID,
		"state":      state,
		"context":    context,
	}
	
	_, err := e.eventBus.Request(e.address, req, 5*time.Second)
	return err
}

func (e *EventBusPersistenceAdapter) Load(ctx context.Context, machineID string) (string, map[string]interface{}, error) {
	req := map[string]interface{}{
		"operation": "load",
		"machineId": machineID,
	}
	
	msg, err := e.eventBus.Request(e.address, req, 5*time.Second)
	if err != nil {
		return "", nil, err
	}
	
	response, ok := msg.Body().(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("invalid response format")
	}
	
	state, _ := response["state"].(string)
	context, _ := response["context"].(map[string]interface{})
	
	return state, context, nil
}

func (e *EventBusPersistenceAdapter) Delete(ctx context.Context, machineID string) error {
	req := map[string]interface{}{
		"operation": "delete",
		"machineId": machineID,
	}
	
	_, err := e.eventBus.Request(e.address, req, 5*time.Second)
	return err
}
