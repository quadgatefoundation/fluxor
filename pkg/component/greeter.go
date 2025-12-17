package component

import (
	"context"
	"log"

	"github.com/fluxor-io/fluxor/pkg/types"
)

// Greeter is a simple component that logs a message on start.
type Greeter struct {
	Base
}

// Name returns the name of the component.
func (g *Greeter) Name() string {
	return "Greeter"
}

// OnStart is a lifecycle hook that is called when the component is started.
func (g *Greeter) OnStart(ctx context.Context, bus types.Bus) error {
	g.Base.OnStart(ctx, bus)
	log.Println("Greeter component started")
	return nil
}
