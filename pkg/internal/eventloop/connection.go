package eventloop

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
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	ww "github.com/lthibault/wetware/pkg"
)

/*
	HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
		   functionality.
*/

const uagentKey = "AgentVersion"

// const (
// 	unidentified identState = iota
// 	identHost
// 	identClient
// )

// type identState uint8

// DispatchNetwork hooks into the host's network and emits events over event.Bus
// to signal changes in connections or streams.
func DispatchNetwork(lx fx.Lifecycle, log log.Logger, host host.Host) error {
	t, err := trackConns(log, host)
	if err != nil {
		return err
	}

	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return t.Close()
		},
	})

	return nil
}

type connTracker struct {
	log log.Logger

	sub                  event.Subscription
	emitConn, emitStream event.Emitter
	m                    peerstore.PeerMetadata

	mu   sync.Mutex
	sync map[peer.ID]chan bool
}

func trackConns(log log.Logger, h host.Host) (t *connTracker, err error) {
	t = &connTracker{
		log:  log,
		m:    h.Peerstore(),
		sync: make(map[peer.ID]chan bool),
	}

	if t.sub, err = h.EventBus().Subscribe([]interface{}{
		new(event.EvtPeerIdentificationCompleted),
		new(event.EvtPeerIdentificationFailed),
	}); err != nil {
		return
	}

	if t.emitConn, err = h.EventBus().Emitter(new(EvtConnectionChanged)); err != nil {
		return
	}

	if t.emitStream, err = h.EventBus().Emitter(new(EvtStreamChanged)); err != nil {
		return
	}

	go func() {
		for v := range t.sub.Out() {
			t.handleIdentEvent(v)
		}
	}()

	h.Network().Notify(&network.NotifyBundle{
		DisconnectedF: t.onDisconnected,

		OpenedStreamF: t.onStreamOpened,
		ClosedStreamF: t.onStreamClosed,
	})

	return
}

func (t *connTracker) Close() error {
	return multierr.Combine(
		t.emitConn.Close(),
		t.emitStream.Close(),
		t.sub.Close(),
	)
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

func (t *connTracker) handleIdentEvent(v interface{}) {
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

	return v.(string) == ww.ClientUAgent
}
