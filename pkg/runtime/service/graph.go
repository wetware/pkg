package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/jitterbug"
	"github.com/lthibault/wetware/pkg/runtime"
	randutil "github.com/lthibault/wetware/pkg/util/rand"
	"go.uber.org/multierr"
)

type (
	// EvtBootRequested is emitted by the Graph service when the local node is orphaned.
	// It signals that out-of-band peer discovery should take place (via the `boot`
	// package).
	EvtBootRequested struct{}

	// EvtGraftRequested is emitted by the Graph service when the local node is not in
	// a "Complete" connectivity phase.  It signals that the peer graph should be
	// queried for supplementary peers.
	EvtGraftRequested struct{}

	// EvtPruneRequested is emitted by the graph service when the local node's
	// connectivity phase is "Overloaded".  It signals that the local node should
	// attempt to terminate connections to remote hosts.
	EvtPruneRequested struct{}
)

// Graph is responsible for maintaining the local node's connectivity properties within
// bounds.
//
// Consumes:
//  - EvtNeighborhoodChanged
//
// Emits:
//	- EvtBootRequested
//  - EvtGraftRequested
//  - EvtPruneRequested
func Graph(h host.Host) ProviderFunc {
	return func() (_ runtime.Service, err error) {

		g := graph{
			bus:       h.EventBus(),
			src:       randutil.FromPeer(h.ID()),
			cq:        make(chan struct{}),
			errs:      make(chan error, 1),
			neighbors: make(chan EvtNeighborhoodChanged),
		}

		if g.tstep, err = g.bus.Subscribe(new(EvtTimestep)); err != nil {
			return
		}

		if g.nhood, err = g.bus.Subscribe(new(EvtNeighborhoodChanged)); err != nil {
			return
		}

		if g.boot, err = g.bus.Emitter(new(EvtBootRequested)); err != nil {
			return
		}

		if g.graft, err = g.bus.Emitter(new(EvtGraftRequested)); err != nil {
			return
		}

		if g.prune, err = g.bus.Emitter(new(EvtPruneRequested)); err != nil {
			return
		}

		return g, nil
	}
}

type graph struct {
	src rand.Source

	cq        chan struct{}
	errs      chan error
	neighbors chan EvtNeighborhoodChanged

	bus                event.Bus
	tstep, nhood       event.Subscription
	boot, graft, prune event.Emitter
}

func (g graph) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "graph",
	}
}

func (g graph) Errors() <-chan error {
	return g.errs
}

func (g graph) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, g.bus); err == nil {
		startBackground(
			g.subloop,
			g.emitloop,
		)
	}

	return
}

func (g graph) Stop(context.Context) error {
	close(g.cq)

	return multierr.Combine(
		g.nhood.Close(),
		g.boot.Close(),
		g.graft.Close(),
		g.prune.Close(),
	)
}

func (g graph) subloop() {
	defer close(g.neighbors)

	var ev EvtNeighborhoodChanged
	sched := newScheduler(time.Minute, jitterbug.Uniform{
		Min:    time.Second * 15,
		Source: rand.New(g.src),
	})

	for {
		select {
		case v, ok := <-g.tstep.Out():
			if !ok {
				return
			}

			if !sched.Advance(v.(EvtTimestep).Delta) {
				continue
			}

			// scheduler deadline reached; reschedule, then re-send `ev` to g.neighbors.
			sched.Reset()
		case v, ok := <-g.nhood.Out():
			if !ok {
				return
			}

			ev = v.(EvtNeighborhoodChanged)
		case <-g.cq:
			return
		}

		select {
		case g.neighbors <- ev:
		case <-g.cq:
			return
		}
	}
}

func (g graph) emitloop() {
	defer close(g.errs)

	for ev := range g.neighbors {
		switch ev.To {
		case PhaseOrphaned:
			g.raise(g.boot.Emit(EvtBootRequested{}))
		case PhasePartial:
			g.raise(g.graft.Emit(EvtGraftRequested{}))
		case PhaseOverloaded:
			g.raise(g.prune.Emit(EvtPruneRequested{}))
		}
	}
}

func (g graph) raise(err error) {
	if err == nil {
		return
	}

	select {
	case g.errs <- err:
	case <-g.cq:
	}
}
