package filter

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"

	"github.com/wetware/ww/pkg/internal/filter"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
)

// Config for Filter service.
type Config struct {
	fx.In

	Bus     event.Bus
	Routing *pubsub.Topic
	Filter  filter.Filter
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (runtime.Service, error) {
	sub, err := cfg.Bus.Subscribe(new(tick_service.EvtTimestep))
	if err != nil {
		return nil, err
	}

	return &router{
		t:   cfg.Filter,
		bus: cfg.Bus,
		rt:  cfg.Routing,
		ts:  sub,
	}, nil
}

// Module for Filter service
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Filter service.  Updates the routing filter, allowing it to "forget" stale peers.
//
// Consumes:
//  - EvtTimestep
//
// Emits:
func New(cfg Config) Module { return Module{Factory: cfg} }

type router struct {
	t  interface{ Advance(time.Time) } // monotonically increasing
	rt *pubsub.Topic
	hb *pubsub.Subscription

	bus event.Bus
	ts  event.Subscription
}

func (r router) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "router",
		"ns":      r.hb.Topic(),
	}
}

func (r *router) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, r.bus); err == nil {
		if r.hb, err = r.rt.Subscribe(); err == nil {
			internal.StartBackground(
				r.tickloop,
				r.sinkloop,
			)
		}
	}

	return
}

func (r router) Stop(context.Context) error {
	r.hb.Cancel()

	return r.ts.Close()
}

func (r router) tickloop() {
	for v := range r.ts.Out() {
		r.t.Advance(v.(tick_service.EvtTimestep).Time)
	}
}

// consumes and discards topic messages
func (r router) sinkloop() {
	for {
		_, err := r.hb.Next(context.Background())
		if err != nil {
			break
		}
	}
}
