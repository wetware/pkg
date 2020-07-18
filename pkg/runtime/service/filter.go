package service

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	log "github.com/lthibault/log/pkg"
	"github.com/lthibault/wetware/pkg/runtime"
)

// Timer is a monotonically increasing function.
type Timer interface {
	Advance(time.Time)
}

// Filter updates the routing filter, allowing it to "forget" stale peers.
//
// Consumes:
//  - EvtTimestep
//
// Emits:
func Filter(bus event.Bus, routing *pubsub.Topic, t Timer) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		r := &router{
			t:    t,
			bus:  bus,
			rt:   routing,
			errs: make(chan error, 1),
		}

		if r.ts, err = bus.Subscribe(new(EvtTimestep)); err != nil {
			return
		}

		return r, nil
	}
}

type router struct {
	t  Timer
	rt *pubsub.Topic
	hb *pubsub.Subscription

	bus event.Bus
	ts  event.Subscription

	errs chan error
}

func (r router) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "router",
		"ns":      r.hb.Topic(),
	}
}

func (r router) Errors() <-chan error {
	return r.errs
}

func (r *router) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, r.bus); err == nil {
		if r.hb, err = r.rt.Subscribe(); err == nil {
			startBackground(
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
		r.t.Advance(v.(EvtTimestep).Time)
	}
}

// consumes and discards topic messages
func (r router) sinkloop() {
	for {
		msg, err := r.hb.Next(context.Background())
		if err != nil {
			break
		}

		log.Info(msg.GetFrom())
	}
}
