package eventloop

import (
	"context"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"
	"go.uber.org/multierr"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	ww "github.com/lthibault/wetware/pkg"
)

/*
	HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
		   functionality.
*/

const uagentKey = "AgentVersion"

// DispatchNetwork hooks into the host's network and emits events over event.Bus
// to signal changes in connections or streams.
func DispatchNetwork(lx fx.Lifecycle, log log.Logger, host host.Host) error {
	on, err := mkNetEmitters(host.EventBus())
	if err != nil {
		return err
	}

	sub, err := host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		return err
	}

	go func() {
		for v := range sub.Out() {
			ev := v.(event.EvtPeerIdentificationCompleted)
			on.Connection.Emit(EvtConnectionChanged{
				Peer:   ev.Peer,
				Client: isClient(log, ev.Peer, host.Peerstore()),
				State:  ConnStateOpened,
			})
		}
	}()

	host.Network().Notify(&network.NotifyBundle{
		// NOTE:  can't use ConnectedF because the
		//		  identity protocol will not have
		// 		  completed, meaning isClient will panic.
		DisconnectedF: onDisconnected(log, on.Connection, host.Peerstore()),

		OpenedStreamF: onStreamOpened(on.Stream),
		ClosedStreamF: onStreamClosed(on.Stream),
	})

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return multierr.Combine(
				on.Connection.Close(),
				on.Stream.Close(),
				sub.Close(),
			)
		},
	})

	return nil
}

func onDisconnected(log log.Logger, e event.Emitter, m peerstore.PeerMetadata) func(network.Network, network.Conn) {
	return func(net network.Network, conn network.Conn) {
		e.Emit(EvtConnectionChanged{
			Peer:   conn.RemotePeer(),
			Client: isClient(log, conn.RemotePeer(), m),
			State:  ConnStateClosed,
		})
	}
}

func onStreamOpened(e event.Emitter) func(network.Network, network.Stream) {
	return func(net network.Network, s network.Stream) {
		e.Emit(EvtStreamChanged{
			Peer:   s.Conn().RemotePeer(),
			Stream: s,
			State:  StreamStateOpened,
		})
	}
}

func onStreamClosed(e event.Emitter) func(network.Network, network.Stream) {
	return func(net network.Network, s network.Stream) {
		e.Emit(EvtStreamChanged{
			Peer:   s.Conn().RemotePeer(),
			Stream: s,
			State:  StreamStateClosed,
		})
	}
}

func mkNetEmitters(bus event.Bus) (s struct{ Connection, Stream event.Emitter }, err error) {
	if s.Connection, err = bus.Emitter(new(EvtConnectionChanged)); err != nil {
		return
	}

	if s.Stream, err = bus.Emitter(new(EvtStreamChanged)); err != nil {
		return
	}

	return
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
			ns=			peer=QmNzXbNoCdWpYKiYKv2VEBVDh21uxoYQ5Pxcck1uxZYzte

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
