package statemachine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// MemoryPersistenceProvider provides in-memory persistence (for testing).
type MemoryPersistenceProvider struct {
	instances map[string]*ExecutionContext
	mu        sync.RWMutex
}

// NewMemoryPersistenceProvider creates a new memory persistence provider.
func NewMemoryPersistenceProvider() *MemoryPersistenceProvider {
	return &MemoryPersistenceProvider{
		instances: make(map[string]*ExecutionContext),
	}
}

// Save saves an instance to memory.
func (p *MemoryPersistenceProvider) Save(instanceID string, execCtx *ExecutionContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Deep copy to avoid mutations
	data, err := json.Marshal(execCtx)
	if err != nil {
		return err
	}
	
	var copy ExecutionContext
	if err := json.Unmarshal(data, &copy); err != nil {
		return err
	}
	
	p.instances[instanceID] = &copy
	return nil
}

// Load loads an instance from memory.
func (p *MemoryPersistenceProvider) Load(instanceID string) (*ExecutionContext, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	execCtx, ok := p.instances[instanceID]
	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}
	
	// Deep copy to avoid mutations
	data, err := json.Marshal(execCtx)
	if err != nil {
		return nil, err
	}
	
	var copy ExecutionContext
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	
	return &copy, nil
}

// Delete deletes an instance from memory.
func (p *MemoryPersistenceProvider) Delete(instanceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.instances, instanceID)
	return nil
}

// List lists all instance IDs for a machine.
func (p *MemoryPersistenceProvider) List(machineID string) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	result := make([]string, 0)
	for id, instance := range p.instances {
		if instance.MachineID == machineID {
			result = append(result, id)
		}
	}
	return result, nil
}

// FilePersistenceProvider provides file-based persistence.
type FilePersistenceProvider struct {
	directory string
	mu        sync.Mutex
}

// NewFilePersistenceProvider creates a new file persistence provider.
func NewFilePersistenceProvider(directory string) (*FilePersistenceProvider, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}
	
	return &FilePersistenceProvider{
		directory: directory,
	}, nil
}

// Save saves an instance to a file.
func (p *FilePersistenceProvider) Save(instanceID string, execCtx *ExecutionContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	filename := filepath.Join(p.directory, instanceID+".json")
	
	data, err := json.MarshalIndent(execCtx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance: %w", err)
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// Load loads an instance from a file.
func (p *FilePersistenceProvider) Load(instanceID string) (*ExecutionContext, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	filename := filepath.Join(p.directory, instanceID+".json")
	
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("instance not found: %s", instanceID)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var execCtx ExecutionContext
	if err := json.Unmarshal(data, &execCtx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}
	
	return &execCtx, nil
}

// Delete deletes an instance file.
func (p *FilePersistenceProvider) Delete(instanceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	filename := filepath.Join(p.directory, instanceID+".json")
	return os.Remove(filename)
}

// List lists all instance IDs for a machine.
func (p *FilePersistenceProvider) List(machineID string) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	entries, err := os.ReadDir(p.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	
	result := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		filename := filepath.Join(p.directory, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		
		var execCtx ExecutionContext
		if err := json.Unmarshal(data, &execCtx); err != nil {
			continue
		}
		
		if execCtx.MachineID == machineID {
			result = append(result, execCtx.InstanceID)
		}
	}
	
	return result, nil
}
