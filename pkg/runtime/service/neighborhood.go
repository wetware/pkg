package service

import (
	"context"
	"fmt"

	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/runtime"
	"go.uber.org/multierr"
)

// EvtNeighborhoodChanged fires when a graph edge is created or destroyed
type EvtNeighborhoodChanged struct {
	// Peer     peer.ID
	K        int
	From, To Phase
}

// Neighborhood produces a neighborhood service.
//
// Consumes:
// 	- event.EvtPeerConnectednessChanged [ libp2p ]
//
// Emits:
//	- EvtNeighborhoodChanged
func Neighborhood(bus event.Bus, kmin, kmax int) ProviderFunc {
	return func() (runtime.Service, error) {
		sub, err := bus.Subscribe(new(event.EvtPeerConnectednessChanged))
		if err != nil {
			return nil, err
		}

		e, err := bus.Emitter(new(EvtNeighborhoodChanged), eventbus.Stateful)
		if err != nil {
			return nil, err
		}

		return neighborhood{
			kmin: kmin,
			kmax: kmax,
			bus:  bus,
			sub:  sub,
			e:    e,
			errs: make(chan error, 1),
		}, nil
	}
}

// neighborhood notifies subscribers of changes in direct connectivity to remote
// hosts.  Neighborhood events do not concern themselves with the number of connections,
// but rather the presence or absence of a direct link.
type neighborhood struct {
	kmin, kmax int
	bus        event.Bus
	sub        event.Subscription
	e          event.Emitter
	errs       chan error
}

func (n neighborhood) Loggable() map[string]interface{} {
	return map[string]interface{}{"service": "neighborhood"}
}

func (n neighborhood) Start(ctx context.Context) (err error) {
	if err = waitNetworkReady(ctx, n.bus); err == nil {
		go n.subloop()

		// signal initial state - PhaseOrphaned
		err = n.e.Emit(EvtNeighborhoodChanged{})
	}

	return
}

func (n neighborhood) Stop(context.Context) error {
	return multierr.Combine(
		n.sub.Close(),
		n.e.Close(),
	)
}

func (n neighborhood) subloop() {
	var ps = make(map[peer.ID]struct{})

	var state EvtNeighborhoodChanged

	for v := range n.sub.Out() {
		switch ev := v.(event.EvtPeerConnectednessChanged); ev.Connectedness {
		case network.Connected:
			ps[ev.Peer] = struct{}{}
		case network.NotConnected:
			delete(ps, ev.Peer)
		default:
			panic("Unreachable ... unless libp2p has fixed event.PeerConnectednessChanged!!")
		}

		state.K = len(ps)
		state.From = state.To
		state.To = n.phase(len(ps))

		if err := n.e.Emit(state); err != nil {
			n.errs <- err
		}
	}
}

func (n neighborhood) phase(k int) Phase {
	switch {
	case k == 0:
		return PhaseOrphaned
	case 0 < k && k < n.kmin:
		return PhasePartial
	case n.kmin < k && k < n.kmax:
		return PhaseComplete
	case k > n.kmax:
		return PhaseOverloaded
	default:
		panic(fmt.Sprintf("invalid cardinality:  %d", k))
	}
}

// Phase is the codomain in the function ƒ: C ⟼ P,
// where C ∈ ℕ and P ∈ {orphaned, partial, complete, overloaded}.  Members of P are
// defined as follows:
//
// Let k ∈ C be the number of remote hosts to which we are connected, and let l, h ∈ ℕ
// be the low-water and high-water marks, respectively.
//
// Then:
// - orphaned := k == 0
// - partial := 0 < k < l
// - complete := l <= k < h
// - overloaded := k >= h
type Phase uint8

const (
	// PhaseOrphaned indicates the Host is not connected to the graph.
	PhaseOrphaned Phase = iota
	// PhasePartial indicates the Host is weakly connected to the graph.
	PhasePartial
	// PhaseComplete indicates the Host is strongly connected to the graph.
	PhaseComplete
	// PhaseOverloaded indicates the Host is strongly connected to the graph, but
	// should have its connections pruned to reduce resource consumption.
	PhaseOverloaded
)

func (p Phase) String() string {
	switch p {
	case PhaseOrphaned:
		return "host orphaned"
	case PhasePartial:
		return "neighborhood partial"
	case PhaseComplete:
		return "neighborhood complete"
	case PhaseOverloaded:
		return "neighborhood overloaded"
	default:
		return fmt.Sprintf("<invalid phase:: %d>", p)
	}
}
