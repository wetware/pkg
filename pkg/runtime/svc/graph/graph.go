package graph

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/lthibault/jitterbug"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/svc/internal"
	"github.com/wetware/ww/pkg/runtime/svc/neighborhood"
	"github.com/wetware/ww/pkg/runtime/svc/ticker"
	randutil "github.com/wetware/ww/pkg/util/rand"
	"go.uber.org/fx"
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

// Config for Graph service.
type Config struct {
	fx.In

	Log  ww.Logger
	Host host.Host
}

// NewService satisfies runtime.ServiceFactory.
func (cfg Config) NewService() (_ runtime.Service, err error) {
	g := graph{
		log:       cfg.Log,
		bus:       cfg.Host.EventBus(),
		src:       randutil.FromPeer(cfg.Host.ID()),
		cq:        make(chan struct{}),
		neighbors: make(chan neighborhood.EvtNeighborhoodChanged),
	}

	if g.tstep, err = g.bus.Subscribe(new(ticker.EvtTimestep)); err != nil {
		return
	}

	if g.nhood, err = g.bus.Subscribe(new(neighborhood.EvtNeighborhoodChanged)); err != nil {
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

// Produces EvtBootRequested, EvtGraftRequested & EvtPruneRequested.
func (cfg Config) Produces() []interface{} {
	return []interface{}{
		EvtBootRequested{},
		EvtGraftRequested{},
		EvtPruneRequested{},
	}
}

// Consumes ticker.EvtTimestep & neighborhood.EvtNeighborhoodChanged.
func (cfg Config) Consumes() []interface{} {
	return []interface{}{
		ticker.EvtTimestep{},
		neighborhood.EvtNeighborhoodChanged{},
	}
}

// Module for Graph service.
type Module struct {
	fx.Out

	Factory runtime.ServiceFactory `group:"runtime"`
}

// New Graph service. Maintainins the local node's connectivity properties within
// bounds.
//
// Consumes:
//  - EvtNeighborhoodChanged
//
// Emits:
//	- EvtBootRequested
//  - EvtGraftRequested
//  - EvtPruneRequested
func New(cfg Config) Module { return Module{Factory: cfg} }

type graph struct {
	log ww.Logger
	src rand.Source

	cq        chan struct{}
	neighbors chan neighborhood.EvtNeighborhoodChanged

	bus                event.Bus
	tstep, nhood       event.Subscription
	boot, graft, prune event.Emitter
}

func (g graph) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "graph",
	}
}

func (g graph) Start(ctx context.Context) (err error) {
	if err = internal.WaitNetworkReady(ctx, g.bus); err == nil {
		internal.StartBackground(
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

	var ev neighborhood.EvtNeighborhoodChanged
	sched := internal.NewScheduler(time.Minute, jitterbug.Uniform{
		Min:    time.Second * 15,
		Source: rand.New(g.src),
	})

	for {
		select {
		case v, ok := <-g.tstep.Out():
			if !ok {
				return
			}

			if !sched.Advance(v.(ticker.EvtTimestep).Delta) {
				continue
			}

			// scheduler deadline reached; reschedule, then re-send `ev` to g.neighbors.
			sched.Reset()
		case v, ok := <-g.nhood.Out():
			if !ok {
				return
			}

			ev = v.(neighborhood.EvtNeighborhoodChanged)
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
	for ev := range g.neighbors {
		switch ev.To {
		case neighborhood.PhaseOrphaned:
			if err := g.boot.Emit(EvtBootRequested{}); err != nil {
				g.log.With(g).WithError(err).Warn("failed to emit EvtBootRequested")
			}

		case neighborhood.PhasePartial:
			if err := g.graft.Emit(EvtGraftRequested{}); err != nil {
				g.log.With(g).WithError(err).Warn("failed to emit EvtGraftRequested")
			}

		case neighborhood.PhaseOverloaded:
			if err := g.prune.Emit(EvtPruneRequested{}); err != nil {
				g.log.With(g).WithError(err).Warn("failed to emit EvtPruneRequested")
			}

		}
	}
}
