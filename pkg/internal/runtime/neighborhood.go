package runtime

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

type neighborhoodTrackerParams struct {
	fx.In

	Log  log.Logger
	Host host.Host
}

// trackNeighbors signals changes in a peer's neighborhood, i.e.:  the set of
// hosts to which it is directly connected.
func trackNeighbors(ctx context.Context, ps neighborhoodTrackerParams, lx fx.Lifecycle) error {
	nt, err := newNeighborhoodTracker(
		ps.Log.WithField("service", "neighborhood"),
		ps.Host.EventBus(),
	)
	if err != nil {
		return err
	}

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go nt.loop()
			nt.log.Debug("service started")
			return nil
		},
		OnStop: func(context.Context) error {
			defer nt.log.Debug("service stopped")
			return nt.Close()
		},
	})

	return nil
}

// neighborhoodTracker notifies subscribers of changes in direct connectivity to remote
// hosts.  Neighborhood events do not concern themselves with the number of connections,
// but rather the presence or absence of a direct link.
type neighborhoodTracker struct {
	log log.Logger
	sub event.Subscription
	e   event.Emitter
}

func newNeighborhoodTracker(log log.Logger, bus event.Bus) (n neighborhoodTracker, err error) {
	n.log = log

	if n.sub, err = bus.Subscribe(new(EvtConnectionChanged)); err != nil {
		return
	}

	if n.e, err = bus.Emitter(new(EvtPeerConnectednessChanged)); err != nil {
		return
	}

	return
}

func (n neighborhoodTracker) Close() error {
	return multierr.Combine(
		n.sub.Close(),
		n.e.Close(),
	)
}

func (n neighborhoodTracker) loop() {
	var (
		emit bool
		ctr  connctr = make(map[peer.ID]int)
	)

	for v := range n.sub.Out() {
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
			n.e.Emit(EvtPeerConnectednessChanged{
				Peer:  ev.Peer,
				State: peerstate(ev.State),
			})
		}
	}
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
