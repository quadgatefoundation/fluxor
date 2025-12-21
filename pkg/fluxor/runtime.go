package fluxor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/khangdcicloud/fluxor/pkg/core"
)

type App struct {
	bus    *core.Bus
	worker *core.WorkerPool
	ctx    context.Context
	cancel context.CancelFunc
}

func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		bus:    core.NewBus(),
		worker: core.NewWorkerPool(10), // Default 10 workers
		ctx:    ctx,
		cancel: cancel,
	}
}

func (a *App) Deploy(c core.Component) {
	id := uuid.New().String()
	fctx := core.NewFluxorContext(a.ctx, a.bus, a.worker, id)
	
	// Start Component in Reactor (Main Thread or Goroutine)
	// Ở đây ta gọi OnStart. Trong mô hình Vert.x, OnStart chạy xong là server đã listen async.
	if err := c.OnStart(fctx); err != nil {
		fmt.Printf("❌ Deploy failed: %v\n", err)
	}
}

func (a *App) Run() {
	fmt.Println("🚀 Fluxor Engine Running... (Ctrl+C to stop)")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\n🛑 Fluxor Shutdown")
	a.cancel()
	a.worker.Shutdown()
}
