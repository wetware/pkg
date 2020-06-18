package runtime

import (
	"context"
	"sync"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"
	"go.uber.org/multierr"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

/*
	HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
		   functionality.
*/

const uagentKey = "AgentVersion"

type connTrackerParams struct {
	fx.In

	Host host.Host
	Log  log.Logger
}

// trackConnections hooks into the host's network and emits events over event.Bus
// to signal changes in connections or streams.
func trackConnections(ctx context.Context, ps connTrackerParams, lx fx.Lifecycle) error {
	t, err := newConnTracker(
		ps.Log.WithField("service", "conntracker"),
		ps.Host.EventBus(),
		ps.Host.Peerstore(),
	)
	if err != nil {
		return err
	}

	ps.Host.Network().Notify(t.notifiee())

	lx.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return t.Start()
		},
		OnStop: func(context.Context) error {
			return t.Close()
		},
	})

	return nil
}

type connTracker struct {
	log log.Logger
	m   peerstore.PeerMetadata

	sub                  event.Subscription
	emitConn, emitStream event.Emitter

	mu   sync.Mutex
	sync map[peer.ID]chan bool
}

func newConnTracker(log log.Logger, bus event.Bus, m peerstore.PeerMetadata) (*connTracker, error) {
	sub, err := bus.Subscribe([]interface{}{
		new(event.EvtPeerIdentificationCompleted),
		new(event.EvtPeerIdentificationFailed),
	})
	if err != nil {
		return nil, err
	}

	emitConn, err := bus.Emitter(new(EvtConnectionChanged))
	if err != nil {
		return nil, err
	}

	emitStream, err := bus.Emitter(new(EvtStreamChanged))
	if err != nil {
		return nil, err
	}

	return &connTracker{
		log:        log,
		m:          m,
		sync:       make(map[peer.ID]chan bool),
		sub:        sub,
		emitConn:   emitConn,
		emitStream: emitStream,
	}, nil
}

func (t *connTracker) Start() error {
	defer t.log.Debug("service started")
	go t.loop()
	return nil
}

func (t *connTracker) Close() error {
	defer t.log.Debug("service stopped")

	return multierr.Combine(
		t.emitConn.Close(),
		t.emitStream.Close(),
		t.sub.Close(),
	)
}

func (t *connTracker) loop() {
	for v := range t.sub.Out() {
		switch ev := v.(type) {
		case event.EvtPeerIdentificationCompleted:
			t.ensureChan(ev.Peer) <- true
			t.emitConn.Emit(EvtConnectionChanged{
				Peer:   ev.Peer,
				Client: t.isClient(ev.Peer),
				State:  ConnStateOpened,
			})
		case event.EvtPeerIdentificationFailed:
			t.ensureChan(ev.Peer) <- false
		}
	}
}

func (t *connTracker) notifiee() network.Notifiee {
	return &network.NotifyBundle{
		DisconnectedF: t.onDisconnected,

		OpenedStreamF: t.onStreamOpened,
		ClosedStreamF: t.onStreamClosed,
	}
}

func (t *connTracker) ensureChan(id peer.ID) chan bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ch, ok := t.sync[id]; ok {
		return ch
	}

	ch := make(chan bool, 1)
	t.sync[id] = ch
	return ch
}

func (t *connTracker) onDisconnected(net network.Network, conn network.Conn) {
	if <-t.ensureChan(conn.RemotePeer()) {
		t.emitConn.Emit(EvtConnectionChanged{
			Peer:   conn.RemotePeer(),
			Client: t.isClient(conn.RemotePeer()),
			State:  ConnStateClosed,
		})
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.sync, conn.RemotePeer())
}

func (t *connTracker) onStreamOpened(net network.Network, s network.Stream) {
	t.emitStream.Emit(EvtStreamChanged{
		Peer:   s.Conn().RemotePeer(),
		Stream: s,
		State:  StreamStateOpened,
	})
}

func (t *connTracker) onStreamClosed(net network.Network, s network.Stream) {
	t.emitStream.Emit(EvtStreamChanged{
		Peer:   s.Conn().RemotePeer(),
		Stream: s,
		State:  StreamStateClosed,
	})
}

// isClient distinguishes between client and host connections using low-level peerstore
// metadata.  This method should not be used outside of the event loop.
//
// The reason it is used here is because remote hosts may not have an entry in the
// filter when they (dis)connect.  This would cause them to be misidentified as clients,
// resuting in an incorrect event being dispatched over the bus.
//
// Developers should prefer to work at the host level, comparing peer.IDs in the
// peerstore to those in the routing table.
func (t *connTracker) isClient(p peer.ID) bool {
	v, err := t.m.Get(p, uagentKey)
	if err != nil {
		panic(err)
	}

	return v.(string) == "ww-client"
}
