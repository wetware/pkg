package service

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/lthibault/wetware/pkg/runtime"
	"go.uber.org/multierr"
)

// Graph is responsible for keeping the local host's connections within acceptable
// bounds.
//
// Consumes:
//  - EvtNeighborhoodChanged
//
// Emits:
//  - EvtPeerDiscovered
func Graph(bus event.Bus, d discovery.Discoverer, ns string, kmax int) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		ctx, cancel := context.WithCancel(context.Background())

		g := graph{
			d:      d,
			ctx:    ctx,
			cancel: cancel,
			errs:   make(chan error, 1),
		}

		if g.sub, err = bus.Subscribe(new(EvtNeighborhoodChanged)); err != nil {
			return
		}

		if g.e, err = bus.Emitter(new(EvtPeerDiscovered)); err != nil {
			return
		}

		return g, nil
	}
}

type graph struct {
	ns   string
	kmax int
	d    discovery.Discoverer

	ctx    context.Context
	cancel context.CancelFunc

	errs chan error
	sub  event.Subscription
	e    event.Emitter
}

func (g graph) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "graph",
	}
}

func (g graph) Errors() <-chan error {
	return g.errs
}

func (g graph) Start(context.Context) (err error) {
	go g.subloop()

	return
}

func (g graph) Stop(context.Context) error {
	defer close(g.errs)
	defer g.cancel()

	return multierr.Combine(
		g.sub.Close(),
		g.e.Close(),
	)
}

func (g graph) subloop() {
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	var ev EvtNeighborhoodChanged

	for {
		select {
		case <-ticker.C:
		case v, ok := <-g.sub.Out():
			if !ok {
				return
			}

			ev = v.(EvtNeighborhoodChanged)
		}

		// NOTE:  the PhaseOrphaned case is handled directly by the Bootstrap service.

		switch ev.To {
		case PhasePartial:
			g.queryDHT(ev)
		case PhaseOverloaded:
			// TODO(enhancement): Trim back some host connections
			g.errs <- errors.New("connection pruning NOT IMPLEMENTED")
		}
	}
}

func (g graph) queryDHT(ev EvtNeighborhoodChanged) {
	ctx, cancel := context.WithTimeout(g.ctx, time.Second*30)
	defer cancel()

	// TODO(bugfix):  provide value for `g.ns` in the DHT (maybe in g.Start?)
	ch, err := g.d.FindPeers(ctx, g.ns, discovery.Limit(g.kmax-ev.K))
	if err != nil {
		g.errs <- err
		return
	}

	for info := range ch {
		if err = g.e.Emit(EvtPeerDiscovered{info}); err != nil {
			g.errs <- err
		}
	}
}
