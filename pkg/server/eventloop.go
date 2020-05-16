package server

/*
	eventloop.go dispatches events on the Host's event bus.  The event bus provides
	asynchronous signals that allow a Host to react to the outside world.
*/

import (
	"context"
	"fmt"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"

	"github.com/lthibault/wetware/pkg/internal/eventloop"
)

type eventLoopConfig struct {
	fx.In

	Log  log.Logger
	Host host.Host
	K    clusterCardinality

	EventHandlers []eventloop.Handler
}

func startEventLoop(lx fx.Lifecycle, cfg eventLoopConfig) (err error) {
	for _, f := range []func(fx.Lifecycle, eventLoopConfig) error{
		registerEventHandlers,
		dispatchNetworkEvts,
		dispatchNeighborhoodEvts,
	} {
		if err = f(lx, cfg); err != nil {
			break
		}
	}

	return
}

func registerEventHandlers(lx fx.Lifecycle, cfg eventLoopConfig) error {
	return eventloop.RegisterHandlers(lx, cfg.Host.EventBus(), cfg.EventHandlers...)
}

func dispatchNetworkEvts(lx fx.Lifecycle, cfg eventLoopConfig) error {
	return eventloop.DispatchNetwork(lx, cfg.Log, cfg.Host)
}

// dispatchNeighborhoodEvts signals changes in a peer's neighborhood, i.e.:  the set of
// hosts to which it is directly connected.
func dispatchNeighborhoodEvts(lx fx.Lifecycle, cfg eventLoopConfig) error {
	bus := cfg.Host.EventBus()

	if err := eventloop.DispatchNeighborhood(lx, bus); err != nil {
		return err
	}

	return startNeighborhood(lx, bus, cfg.K)
}

func startNeighborhood(lx fx.Lifecycle, bus event.Bus, k clusterCardinality) error {
	sub, err := bus.Subscribe(new(eventloop.EvtPeerConnectednessChanged))
	if err != nil {
		return err
	}
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	e, err := bus.Emitter(new(eventloop.EvtNeighborhoodChanged))
	if err != nil {
		return err
	}

	go neighborhoodEventLoop(sub, e, k)
	return nil
}

func neighborhoodEventLoop(sub event.Subscription, e event.Emitter, k clusterCardinality) {
	defer e.Close()

	var (
		out   eventloop.EvtNeighborhoodChanged
		phase = phaseTracker(k)
	)

	for v := range sub.Out() {
		ev := v.(eventloop.EvtPeerConnectednessChanged)

		switch ev.State {
		case eventloop.PeerStateConnected:
			out.N++
		case eventloop.PeerStateDisconnected:
			out.N--
		default:
			panic(fmt.Sprintf("unknown peer state %d", ev.State))
		}

		out.Peer = ev.Peer
		out.From = out.To
		out.To = phase(out.N)

		e.Emit(out)
	}
}

func phaseTracker(k clusterCardinality) func(int) eventloop.Phase {
	return func(n int) eventloop.Phase {
		switch {
		case n < 0:
			return eventloop.PhaseOrphaned
		case n < k.Min:
			return eventloop.PhasePartial
		case n < k.Max:
			return eventloop.PhaseComplete
		case n >= k.Max:
			return eventloop.PhaseOverloaded
		default:
			panic(fmt.Sprintf("invalid number of connections: %d", k))
		}
	}
}
