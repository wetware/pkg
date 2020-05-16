package eventloop

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/fx"
)

/*
	neighborhood.go dispatches events that track immediate connections to peers.
*/

// DispatchNeighborhood notifies subscribers of changes in direct connectivity to remote
// hosts.  Neighborhood events do not concern themselves with the number of connections,
// but rather the presence or absence of a direct link.
func DispatchNeighborhood(lx fx.Lifecycle, bus event.Bus) error {
	sub, err := bus.Subscribe(new(EvtConnectionChanged))
	if err != nil {
		return err
	}

	e, err := bus.Emitter(new(EvtPeerConnectednessChanged))
	if err != nil {
		return err
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	go func() {
		defer e.Close()

		var (
			emit bool
			ctr  connctr = make(map[peer.ID]int)
		)

		for v := range sub.Out() {
			ev := v.(EvtConnectionChanged)
			if ev.Client {
				continue
			}

			switch ev.State {
			case ConnStateOpened:
				emit = ctr.Add(ev.Peer)
			case ConnStateClosed:
				emit = ctr.Rm(ev.Peer)
			default:
				panic(fmt.Sprintf("unknown conn state %d", ev.State))
			}

			if emit {
				e.Emit(EvtPeerConnectednessChanged{
					Peer:  ev.Peer,
					State: peerstate(ev.State),
				})
			}
		}
	}()

	return nil
}

type connctr map[peer.ID]int

func (ctr connctr) Add(id peer.ID) (leased bool) {
	i, ok := ctr[id]
	if !ok {
		leased = true
	}

	ctr[id] = i + 1
	return
}

func (ctr connctr) Rm(id peer.ID) (evicted bool) {
	i, ok := ctr[id]
	if !ok {
		// if we ever hit this (and it's _actually_ isClient), consider simply removing
		// this test and returning false.
		panic("unreachable - probably caused by isClient")
	}

	if i == 1 {
		delete(ctr, id)
		evicted = true
	} else {
		ctr[id] = i - 1
	}

	return
}

func peerstate(cs ConnState) PeerState {
	switch cs {
	case ConnStateOpened:
		return PeerStateConnected
	case ConnStateClosed:
		return PeerStateDisconnected
	default:
		panic(fmt.Sprintf("unrecognized ConnState %d", cs))
	}
}
