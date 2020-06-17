package eventloop

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

// TODO:  consider moving this to package `server`.  Consider also making private.

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

	// EvtPeerConnectednessChanged fires when a direct connection to a remote host is
	// established or lost.  This is distinct from EvtConnectionChanged in two ways:
	//
	//  1. It is concerned with the presence / absence of connections, not the quantity.
	//  2. It is emitted only for host connections.  Clients are ignored.
	//
	// Developers should generally prefer EvtPeerConnectednessChanged over
	// EvtConnectionChanged.
	EvtPeerConnectednessChanged struct {
		Peer  peer.ID
		State PeerState
	}

	// EvtNeighborhoodChanged fires when a graph edge is created or destroyed
	EvtNeighborhoodChanged struct {
		Peer     peer.ID
		From, To Phase
		N        int
	}
)

// Phase is the codomain in the function ƒ: C ⟼ P,
// where C ∈ ℕ and P ∈ {orphaned, partial, complete, overloaded}.  Members of P are
// defined as follows:
//
// Let n ∈ C be the number of active connections to remote hosts (i.e., excluding client
// connections), and l, h ∈ ℕ be the low-water and high-water marks, respectively.
//
// Then:
// - orphaned := n == 0
// - partial := 0 < n < l
// - complete := l <= n < h
// - overloaded := n >= h
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
		return "orphaned"
	case PhasePartial:
		return "partial"
	case PhaseComplete:
		return "complete"
	case PhaseOverloaded:
		return "overloaded"
	default:
		return fmt.Sprintf("<invalid :: %d>", p)
	}
}

// ConnState tags a connection as belonging to a client or server.
type ConnState uint8

const (
	// ConnStateOpened .
	ConnStateOpened ConnState = iota

	// ConnStateClosed .
	ConnStateClosed
)

func (c ConnState) String() string {
	switch c {
	case ConnStateOpened:
		return "opened"
	case ConnStateClosed:
		return "closed"
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
		return "opened"
	case StreamStateClosed:
		return "closed"
	default:
		return fmt.Sprintf("<invalid :: %d>", s)
	}
}

// PeerState tags a peer state
type PeerState uint8

const (
	// PeerStateConnected .
	PeerStateConnected PeerState = iota

	// PeerStateDisconnected .
	PeerStateDisconnected
)

func (s PeerState) String() string {
	switch s {
	case PeerStateConnected:
		return "opened"
	case PeerStateDisconnected:
		return "closed"
	default:
		return fmt.Sprintf("<invalid :: %d>", s)
	}
}
