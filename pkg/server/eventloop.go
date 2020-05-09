package server

/*
	eventloop.go dispatches events on the Host's event bus.  The event bus provides
	asynchronous signals that allow a Host to react to the outside world.
*/

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	log "github.com/lthibault/log/pkg"
	ww "github.com/lthibault/wetware/pkg"
)

const uagentKey = "AgentVersion"

type evtLoopConfig struct {
	fx.In

	Log  log.Logger
	Host host.Host
	KMin int `name:"kmin"`
	KMax int `name:"kmax"`

	EventHandlers []evtHandler
}

// main event loop
func eventloop(lx fx.Lifecycle, cfg evtLoopConfig) (err error) {
	for _, f := range []func(fx.Lifecycle, evtLoopConfig) error{
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

// dispatchNetworkEvts hooks into the host's network and emits events over event.Bus to
// signal changes in connections or streams.
//
// HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
//		  functionality.
func dispatchNetworkEvts(lx fx.Lifecycle, cfg evtLoopConfig) error {
	on, err := mkNetEmitters(cfg.Host.EventBus())
	if err != nil {
		return err
	}

	sub, err := cfg.Host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		return err
	}

	go func() {
		for v := range sub.Out() {
			ev := v.(event.EvtPeerIdentificationCompleted)
			on.Connection.Emit(ww.EvtConnectionChanged{
				Peer:   ev.Peer,
				Client: isClient(cfg.Log, ev.Peer, cfg.Host.Peerstore()),
				State:  ww.ConnStateOpened,
			})
		}
	}()

	cfg.Host.Network().Notify(&network.NotifyBundle{
		// NOTE:  can't use ConnectedF because the
		//		  identity protocol will not have
		// 		  completed, meaning isClient will panic.
		DisconnectedF: onDisconnected(cfg.Log, on.Connection, cfg.Host.Peerstore()),

		OpenedStreamF: onStreamOpened(on.Stream),
		ClosedStreamF: onStreamClosed(on.Stream),
	})

	lx.Append(fx.Hook{
		OnStop: func(context.Context) (err error) {
			for _, c := range []io.Closer{on.Connection, on.Stream, sub} {
				if e := c.Close(); e != nil && err == nil {
					err = e
				}
			}

			return
		},
	})

	return nil
}

func onDisconnected(log log.Logger, e event.Emitter, m peerstore.PeerMetadata) func(network.Network, network.Conn) {
	return func(net network.Network, conn network.Conn) {
		e.Emit(ww.EvtConnectionChanged{
			Peer:   conn.RemotePeer(),
			Client: isClient(log, conn.RemotePeer(), m),
			State:  ww.ConnStateClosed,
		})
	}
}

func onStreamOpened(e event.Emitter) func(network.Network, network.Stream) {
	return func(net network.Network, s network.Stream) {
		e.Emit(ww.EvtStreamChanged{
			Peer:   s.Conn().RemotePeer(),
			Stream: s,
			State:  ww.StreamStateOpened,
		})
	}
}

func onStreamClosed(e event.Emitter) func(network.Network, network.Stream) {
	return func(net network.Network, s network.Stream) {
		e.Emit(ww.EvtStreamChanged{
			Peer:   s.Conn().RemotePeer(),
			Stream: s,
			State:  ww.StreamStateClosed,
		})
	}
}

func mkNetEmitters(bus event.Bus) (s struct{ Connection, Stream event.Emitter }, err error) {
	if s.Connection, err = bus.Emitter(new(ww.EvtConnectionChanged)); err != nil {
		return
	}

	if s.Stream, err = bus.Emitter(new(ww.EvtStreamChanged)); err != nil {
		return
	}

	return
}

func dispatchNeighborhoodEvts(lx fx.Lifecycle, cfg evtLoopConfig) error {
	var sub event.Subscription
	var n = newNeighborhood(cfg.KMin, cfg.KMax)

	sub, err := cfg.Host.EventBus().Subscribe(new(ww.EvtConnectionChanged))
	if err != nil {
		return err
	}
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return sub.Close()
		},
	})

	e, err := cfg.Host.EventBus().Emitter(new(ww.EvtNeighborhoodChanged))
	if err != nil {
		return err
	}

	go neighborhoodEventLoop(sub, e, n)
	return nil
}

func neighborhoodEventLoop(sub event.Subscription, e event.Emitter, n neighborhood) {
	defer e.Close()

	var (
		fire bool
		out  ww.EvtNeighborhoodChanged
	)

	for v := range sub.Out() {
		ev := v.(ww.EvtConnectionChanged)
		if ev.Client {
			continue
		}

		switch ev.State {
		case ww.ConnStateOpened:
			fire = n.Add(ev.Peer)
		case ww.ConnStateClosed:
			fire = n.Rm(ev.Peer)
		default:
			panic(fmt.Sprintf("unknown conn state %d", ev.State))
		}

		if fire {
			out = ww.EvtNeighborhoodChanged{
				Peer:  ev.Peer,
				State: ev.State,
				From:  out.To,
				To:    n.Phase(),
				N:     n.Len(),
			}

			e.Emit(out)
		}
	}
}

type neighborhood struct {
	kmin, kmax int
	m          map[peer.ID]int
}

func newNeighborhood(kmin, kmax int) neighborhood {
	return neighborhood{
		kmin: kmin,
		kmax: kmax,
		m:    make(map[peer.ID]int),
	}
}

func (n neighborhood) Len() int {
	return len(n.m)
}

func (n neighborhood) Add(id peer.ID) (leased bool) {
	i, ok := n.m[id]
	if !ok {
		leased = true
	}

	n.m[id] = i + 1
	return
}

func (n neighborhood) Rm(id peer.ID) (evicted bool) {
	i, ok := n.m[id]
	if !ok {
		// if we ever hit this (and it's _actually_ isClient), consider simply removing
		// this test and returning false.
		panic("unreachable - probably caused by isClient")
	}

	if i == 1 {
		delete(n.m, id)
		evicted = true
	} else {
		n.m[id] = i - 1
	}

	return
}

func (n neighborhood) Phase() ww.Phase {
	switch k := len(n.m); {
	case k < 0:
		return ww.PhaseOrphaned
	case k < n.kmin:
		return ww.PhasePartial
	case k < n.kmax:
		return ww.PhaseComplete
	case k >= n.kmax:
		return ww.PhaseOverloaded
	default:
		panic(fmt.Sprintf("invalid number of connections: %d", k))
	}
}

// isClient distinguishes between client and host connections using low-level peerstore
// metadata.  This method should not be used outside of the event loop.
//
// The reason it is used here is because remote hosts may not have an entry in the
// filter when they (dis)connect.  This would cause them to be misidentified as clients,
// resuting in an incorrect event being dispatched over the bus.
//
// Developers should prefer to work at the host level, comparing peer.IDs in the
// peerstore to those in the filter/routing-table by means of `filter.Contains`.
func isClient(log log.Logger, p peer.ID, ps peerstore.PeerMetadata) bool {
	switch v, err := ps.Get(p, uagentKey); err {
	case nil:
		return v.(string) == ww.ClientUAgent
	case peerstore.ErrNotFound:
		// This usually means isClient was called in network.Notifiee.Connected, before
		// authentication by the IDService completed.

		// XXX: this is stochastically triggered with the following log output appearing
		//      immediately before the panic stack-trace appears.
		//
		//      Best guess:
		//			1.  connection established
		//			2.  id stream opened
		//			3.  something goes wrong ==> stream reset ==> conn closed
		//			4.  onConnClosed triggered, but event.PeerIdentificationCompleted
		//				never fired ==> user agent not present ==> panic.
		//
		//		MITIGATION:  emit error-level log message instead of panicking.

		/*
			TRAC[0001] peer identification failed

			addrs="[/ip4/127.0.0.1/tcp/64725 /ip6/::1/tcp/64726]"
			error="stream reset"
			id=QmXJjG9TZzrmQV419v2vcVoGuv15U3RreYTb1b8js9S2id
			ns=ww
			peer=QmNzXbNoCdWpYKiYKv2VEBVDh21uxoYQ5Pxcck1uxZYzte

			panic: user agent not found: item not found
		*/

		log.WithError(err).Error("isClient failed to get user agent")

		// HACK: so far we've only observed this in host-host connections (though we've
		// never tested a host-client conn).
		//
		// False positives (clients misidentified as hosts) will trigger a panic in
		// neighborhood.Rm.  If this happens, consider simply removing the panic
		// statement in `Rm`.  We _probably_ don't need it.
		//
		// Either way, this is a tempororary hack until upstream libp2p begins emitting
		// event.PeerConnectednessChanged.
		return true
	default:
		panic(err)
	}
}

// event handlers

type evtHandler struct {
	ev  interface{}
	cb  func(interface{})
	opt []event.SubscriptionOpt
}

func registerEventHandlers(lx fx.Lifecycle, cfg evtLoopConfig) (err error) {
	ss := make([]event.Subscription, len(cfg.EventHandlers))
	for i, h := range cfg.EventHandlers {
		if ss[i], err = cfg.Host.EventBus().Subscribe(h.ev, h.opt...); err != nil {
			return
		}

		go watchEventChan(ss[i].Out(), h.cb)
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			var g errgroup.Group
			for _, sub := range ss {
				g.Go(sub.Close)
			}
			return g.Wait()
		},
	})

	return
}

func watchEventChan(events <-chan interface{}, f func(interface{})) {
	for event := range events {
		f(event)
	}
}
