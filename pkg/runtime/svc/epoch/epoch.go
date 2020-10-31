package epoch

import (
	"context"

	"github.com/libp2p/go-libp2p-core/event"
	"go.uber.org/fx"

	"github.com/wetware/ww/pkg/cluster"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	"github.com/wetware/ww/pkg/runtime/svc/ticker"
)

// Config for Filter service.
type Config struct {
	fx.In

	Bus   event.Bus
	Epoch cluster.EpochController
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (runtime.Service, error) {
	sub, err := cfg.Bus.Subscribe(new(ticker.EvtTimestep))
	if err != nil {
		return nil, err
	}

	return &router{
		epoch: cfg.Epoch,
		bus:   cfg.Bus,
		ts:    sub,
	}, nil
}

// Consumes ticker.EvtTimestep.
func (cfg Config) Consumes() []interface{} {
	return []interface{}{
		ticker.EvtTimestep{},
	}
}

// Module for Filter service
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Filter service.  Updates the routing filter, allowing it to "forget" stale peers.
func New(cfg Config) Module { return Module{Factory: cfg} }

type router struct {
	epoch cluster.EpochController

	bus event.Bus
	ts  event.Subscription
}

func (r router) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "router",
	}
}

func (r *router) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, r.bus); err == nil {
		go r.tickloop()
	}

	return
}

func (r router) Stop(context.Context) error { return r.ts.Close() }

func (r router) tickloop() {
	for v := range r.ts.Out() {
		r.epoch.Advance(v.(ticker.EvtTimestep).Time)
	}
}
