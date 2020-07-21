package service

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/multierr"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/wetware/ww/pkg/runtime"
)

/*
	HACK:  This is a short-term solution while we wait for libp2p to provide equivalent
		   functionality.
*/

type (
	// EvtConnectionChanged fires when a peer connection has been created or destroyed.
	EvtConnectionChanged struct {
		Peer   peer.ID
		State  ConnState
		Client bool
	}

	// EvtStreamChanged fires when a stream is opened or closed.
	EvtStreamChanged struct {
		Peer   peer.ID
		State  StreamState
		Stream interface {
			Protocol() protocol.ID
		}
	}
)

// conntracker emits events whenever connections are created or destroyed.
type conntracker struct {
	m peerstore.PeerMetadata

	idsub, connsub                 event.Subscription
	emitConn, emitStream, emitPeer event.Emitter

	mu   sync.Mutex
	sync map[peer.ID]chan bool
	cq   chan struct{}
}

// ConnTracker produces a conntracker service.
//
// consumes:
// 	- event.EvtPeerIdentificationCompleted  [ libp2p ]
//  - event.EvtPeerIdentificationFailed     [ libp2p ]
//
// emits:
//  - EvtPeerConnectednessChanged [ libp2p ]
//  - EvtConnectionChanged
//  - EvtStreamChanged
func ConnTracker(h host.Host) ProviderFunc {
	return func() (_ runtime.Service, err error) {
		t := &conntracker{
			m:    h.Peerstore(),
			sync: make(map[peer.ID]chan bool),
			cq:   make(chan struct{}),
		}

		bus := h.EventBus()

		//
		// Stream and connection counting
		if t.idsub, err = bus.Subscribe([]interface{}{
			new(event.EvtPeerIdentificationCompleted),
			new(event.EvtPeerIdentificationFailed),
		}); err != nil {
			return
		}

		if t.emitConn, err = bus.Emitter(new(EvtConnectionChanged)); err != nil {
			return
		}

		if t.emitStream, err = bus.Emitter(new(EvtStreamChanged)); err != nil {
			return
		}

		//
		// Peer connectedness
		if t.connsub, err = bus.Subscribe(new(EvtConnectionChanged)); err != nil {
			return
		}

		if t.emitPeer, err = bus.Emitter(new(event.EvtPeerConnectednessChanged)); err != nil {
			return
		}

		h.Network().Notify(t.notifiee())

		return t, nil
	}
}

// Loggable representation of conntracker
func (t *conntracker) Loggable() map[string]interface{} {
	return map[string]interface{}{"service": "conntracker"}
}

// Start service
func (t *conntracker) Start(context.Context) error {
	startBackground(
		t.idloop,
		t.connloop,
	)

	return nil
}

// Stop service
func (t *conntracker) Stop(context.Context) error {
	close(t.cq)

	return multierr.Combine(
		t.emitConn.Close(),
		t.emitStream.Close(),
		t.idsub.Close(),
	)
}

func (t *conntracker) idloop() {
	for v := range t.idsub.Out() {
		switch ev := v.(type) {
		case event.EvtPeerIdentificationCompleted:
			t.ensureChan(ev.Peer) <- true // buffered; nonblocking
			t.emitConn.Emit(EvtConnectionChanged{
				Peer:   ev.Peer,
				Client: t.isClient(ev.Peer),
				State:  ConnStateOpened,
			})
		case event.EvtPeerIdentificationFailed:
			t.ensureChan(ev.Peer) <- false // buffered; nonblocking
		}
	}
}

func (t *conntracker) connloop() {
	var emit bool
	var ctr connctr = make(map[peer.ID]int)

	for v := range t.connsub.Out() {
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
			t.emitPeer.Emit(event.EvtPeerConnectednessChanged{
				Peer:          ev.Peer,
				Connectedness: peerstate(ev.State),
			})
		}
	}
}

func (t *conntracker) notifiee() network.Notifiee {
	return &network.NotifyBundle{
		DisconnectedF: t.onDisconnected,

		OpenedStreamF: t.onStreamOpened,
		ClosedStreamF: t.onStreamClosed,
	}
}

func (t *conntracker) ensureChan(id peer.ID) chan bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ch, ok := t.sync[id]; ok {
		return ch
	}

	ch := make(chan bool, 1)
	t.sync[id] = ch
	return ch
}

func (t *conntracker) onDisconnected(net network.Network, conn network.Conn) {
	select {
	case ok := <-t.ensureChan(conn.RemotePeer()):
		if ok {
			t.emitConn.Emit(EvtConnectionChanged{
				Peer:   conn.RemotePeer(),
				Client: t.isClient(conn.RemotePeer()),
				State:  ConnStateClosed,
			})
		}

		t.mu.Lock()
		defer t.mu.Unlock()
		delete(t.sync, conn.RemotePeer())
	case <-t.cq:
	}
}

func (t *conntracker) onStreamOpened(net network.Network, s network.Stream) {
	t.emitStream.Emit(EvtStreamChanged{
		Peer:   s.Conn().RemotePeer(),
		Stream: s,
		State:  StreamStateOpened,
	})
}

func (t *conntracker) onStreamClosed(net network.Network, s network.Stream) {
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
func (t *conntracker) isClient(p peer.ID) bool {
	v, err := t.m.Get(p, uagentKey)
	if err != nil {
		panic(err)
	}

	return v.(string) == "ww-client"
}

// ConnState tags a connection as belonging to a client or server.
type ConnState uint8

const (
	uagentKey = "AgentVersion"

	// ConnStateOpened .
	ConnStateOpened ConnState = iota

	// ConnStateClosed .
	ConnStateClosed
)

func (c ConnState) String() string {
	switch c {
	case ConnStateOpened:
		return "connection opened"
	case ConnStateClosed:
		return "connection closed"
	default:
		return fmt.Sprintf("<invalid :: %d>", c)
	}
}

// StreamState tags a stream state.
type StreamState uint8

const (
	// StreamStateOpened .
	StreamStateOpened StreamState = iota

	// StreamStateClosed .
	StreamStateClosed
)

func (s StreamState) String() string {
	switch s {
	case StreamStateOpened:
		return "stream opened"
	case StreamStateClosed:
		return "stream closed"
	default:
		return fmt.Sprintf("<invalid :: %d>", s)
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

func peerstate(cs ConnState) network.Connectedness {
	switch cs {
	case ConnStateOpened:
		return network.Connected
	case ConnStateClosed:
		return network.NotConnected
	default:
		panic(fmt.Sprintf("unrecognized ConnState %d", cs))
	}
}
