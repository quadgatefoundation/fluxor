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

// Status represents a snapshot of the runtime's state.
type Status struct {
	State      string
	Components []string
	Reactors   []string
}

type Runtime struct {
	bus         types.Bus
	state       uint32
	comps       map[string]types.Component
	mu          sync.RWMutex
	reactors    map[string]*reactor.Reactor
	workerPool  *worker.WorkerPool
	mailboxSize int
}

func NewRuntime(opts Options) *Runtime {
	return &Runtime{
		bus:         bus.NewBus(),
		comps:       make(map[string]types.Component),
		reactors:    make(map[string]*reactor.Reactor),
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
	compReactor := reactor.New(r.mailboxSize)
	r.reactors[name] = compReactor
	r.comps[name] = comp

	if atomic.LoadUint32(&r.state) == runtimeStateStarted {
		// If the runtime is already running, start the new component immediately.
		compReactor.Start()
		go func() {
			if err := comp.OnStart(ctx, r.bus); err != nil {
				// A real implementation should have a better error handling strategy
			}
		}()
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
		compReactor := r.reactors[name]
		compReactor.Start()
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
			r.reactors[name].Stop(ctx)
		}(name, comp)
	}
	wg.Wait()
	r.mu.RUnlock()

	r.workerPool.Stop(ctx)

	atomic.StoreUint32(&r.state, runtimeStateStopped)
	return nil
}

func (r *Runtime) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stateMap := map[uint32]string{
		runtimeStateIdle:     "Idle",
		runtimeStateStarting: "Starting",
		runtimeStateStarted:  "Started",
		runtimeStateStopping: "Stopping",
		runtimeStateStopped:  "Stopped",
	}

	comps := make([]string, 0, len(r.comps))
	for name := range r.comps {
		comps = append(comps, name)
	}

	var reactors []string
	for name := range r.reactors {
		reactors = append(reactors, name)
	}

	return Status{
		State:      stateMap[atomic.LoadUint32(&r.state)],
		Components: comps,
		Reactors:   reactors,
	}
}

func (r *Runtime) Bus() types.Bus {
	return r.bus
}
