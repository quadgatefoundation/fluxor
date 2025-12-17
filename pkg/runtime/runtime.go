package runtime

import (
	"context"
	"fmt"
	"github.com/example/goreactor/pkg/reactor"
	"hash/fnv"
	"sync"
)

type Component interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Runtime struct {
	numReactors int
	reactors    []*reactor.Reactor
	components  map[string]Component
	mu          sync.RWMutex
}

func NewRuntime(numReactors int) *Runtime {
	if numReactors <= 0 {
		numReactors = 1
	}

	rt := &Runtime{
		numReactors: numReactors,
		reactors:    make([]*reactor.Reactor, numReactors),
		components:  make(map[string]Component),
	}

	for i := 0; i < numReactors; i++ {
		rt.reactors[i] = reactor.NewReactor(reactor.ReactorOptions{})
	}

	return rt
}

func (rt *Runtime) Start() {
	for _, r := range rt.reactors {
		r.Start()
	}
}

func (rt *Runtime) Stop(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, r := range rt.reactors {
		wg.Add(1)
		go func(r *reactor.Reactor) {
			defer wg.Done()
			r.Stop(ctx)
		}(r)
	}
	wg.Wait()
	return nil
}

func (rt *Runtime) Deploy(ctx context.Context, c Component) (string, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	id := fmt.Sprintf("%p", c)
	rt.components[id] = c

	r := rt.ReactorForKey(id)
	r.Post(func() {
		c.Start(ctx)
	})

	return id, nil
}

func (rt *Runtime) Undeploy(ctx context.Context, id string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	c, ok := rt.components[id]
	if !ok {
		return fmt.Errorf("component not found: %s", id)
	}

	delete(rt.components, id)

	r := rt.ReactorForKey(id)
	r.Post(func() {
		c.Stop(ctx)
	})

	return nil
}

func (rt *Runtime) ReactorForKey(key string) *reactor.Reactor {
	h := fnv.New32a()
	h.Write([]byte(key))
	return rt.reactors[h.Sum32()%uint32(rt.numReactors)]
}
