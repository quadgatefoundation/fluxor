package runtime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/reactor"
	"github.com/fluxor-io/fluxor/pkg/types"
	"github.com/fluxor-io/fluxor/pkg/worker"
)

var ErrRuntimeAlreadyStarted = errors.New("runtime has already been started")
var ErrRuntimeNotStarted = errors.New("runtime is not started")

const (
	runtimeStateIdle uint32 = iota
	runtimeStateStarting
	runtimeStateStarted
	runtimeStateStopping
	runtimeStateStopped
)

type Options struct {
	MailboxSize int
	NumWorkers  int
	QueueSize   int
}

type Runtime struct {
	bus         bus.Bus
	state       uint32
	comps       map[string]types.Component
	mu          sync.RWMutex
	reactors    *bus.ReactorStore
	workerPool  *worker.WorkerPool
	mailboxSize int
}

func NewRuntime(opts Options) *Runtime {
	busImpl := bus.NewBus()
	reactors := bus.NewReactorStore()
	busImpl.SetReactorProvider(reactors)
	return &Runtime{
		bus:         busImpl,
		comps:       make(map[string]types.Component),
		reactors:    reactors,
		workerPool:  worker.NewWorkerPool(opts.NumWorkers, opts.QueueSize),
		mailboxSize: opts.MailboxSize,
	}
}

func (r *Runtime) Deploy(ctx context.Context, comp types.Component) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := comp.Name()
	if _, exists := r.comps[name]; exists {
		return errors.New("component already deployed: " + name)
	}

	// Each component gets its own reactor, enforcing the actor model.
	compReactor := reactor.NewReactor(name, r.mailboxSize)
	r.reactors.AddReactor(name, compReactor)
	r.comps[name] = comp

	if atomic.LoadUint32(&r.state) == runtimeStateStarted {
		// If the runtime is already running, start the new component immediately.
		return compReactor.OnStart(ctx, r.bus)
	}

	return nil
}

func (r *Runtime) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.state, runtimeStateIdle, runtimeStateStarting) {
		return ErrRuntimeAlreadyStarted
	}

	r.workerPool.Start()

	r.mu.RLock()
	for name, comp := range r.comps {
		reactor, _ := r.reactors.GetReactor(name)
		if err := reactor.OnStart(ctx, r.bus); err != nil {
			// In a real-world scenario, we might want to stop already started reactors.
			r.mu.RUnlock()
			return err
		}
		if err := comp.OnStart(ctx, r.bus); err != nil {
			r.mu.RUnlock()
			return err
		}
	}
	r.mu.RUnlock()

	atomic.StoreUint32(&r.state, runtimeStateStarted)
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.state, runtimeStateStarted, runtimeStateStopping) {
		return ErrRuntimeNotStarted
	}

	var wg sync.WaitGroup
	r.mu.RLock()
	wg.Add(len(r.comps))
	for name, comp := range r.comps {
		go func(name string, comp types.Component) {
			defer wg.Done()
			comp.OnStop(ctx) // Errors are logged within the component.
			reactor, _ := r.reactors.GetReactor(name)
			reactor.OnStop(ctx)
		}(name, comp)
	}
	wg.Wait()
	r.mu.RUnlock()

	r.workerPool.Stop(ctx)

	atomic.StoreUint32(&r.state, runtimeStateStopped)
	return nil
}

func (r *Runtime) Bus() types.Bus {
	return r.bus
}
