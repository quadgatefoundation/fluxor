package bus

import (
	"sync"

	"github.com/fluxor-io/fluxor/pkg/types"
)

// ReactorProvider defines the interface for providing reactors.
// This is used by the bus to look up the correct reactor for a component.

type ReactorProvider interface {
	GetReactor(componentName string) (types.Reactor, bool)
}

// ReactorStore is a concrete implementation of ReactorProvider that stores reactors in a map.

type ReactorStore struct {
	mu       sync.RWMutex
	reactors map[string]types.Reactor
}

// NewReactorStore creates a new ReactorStore.
func NewReactorStore() *ReactorStore {
	return &ReactorStore{
		reactors: make(map[string]types.Reactor),
	}
}

// AddReactor adds a reactor to the store.
func (s *ReactorStore) AddReactor(name string, reactor types.Reactor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reactors[name] = reactor
}

// GetReactor retrieves a reactor from the store.
func (s *ReactorStore) GetReactor(name string) (types.Reactor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reactor, found := s.reactors[name]
	return reactor, found
}
