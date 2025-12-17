package bus

import (
	"sync"

	"github.com/fluxor-io/fluxor/pkg/reactor"
)

// ReactorProvider provides a way to get a reactor for a given component.
type ReactorProvider interface {
	GetReactor(componentName string) (*reactor.Reactor, bool)
}

// reactorStore implements the ReactorProvider interface.
type reactorStore struct {
	reactors map[string]*reactor.Reactor
	mu       sync.RWMutex
}

// NewReactorStore creates a new reactorStore.
func NewReactorStore() *reactorStore {
	return &reactorStore{
		reactors: make(map[string]*reactor.Reactor),
	}
}

// AddReactor adds a reactor to the store.
func (s *reactorStore) AddReactor(componentName string, r *reactor.Reactor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reactors[componentName] = r
}

// GetReactor returns a reactor for a given component.
func (s *reactorStore) GetReactor(componentName string) (*reactor.Reactor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.reactors[componentName]
	return r, ok
}
