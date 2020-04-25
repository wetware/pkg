package ww

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

const (
	// DefaultNamespace .
	DefaultNamespace = "ww"

	// LowWater is the minimum number of desired connections in a host's resting state.
	// If a host has n < LowWater connections, it will periodically attempt to discover
	// and connect to new peers.
	LowWater = 8

	// HighWater is the maximum number of desired connections in a host's resting state.
	// If a host has n > HighWater connections, it will attempt to close connections
	// until LowWater <= n <= HighWater.
	HighWater = 32

	// ClientUAgent is the user agent for a client connection.
	ClientUAgent = "ww-client"
)

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

	// EvtNeighborhoodChanged fires when a graph edge is created or destroyed
	EvtNeighborhoodChanged struct {
		Peer     peer.ID
		State    ConnState
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

// ConnState tags a connection as belonging to a client or server.
type ConnState uint8

const (
	// ConnStateOpened .
	ConnStateOpened ConnState = iota

	// ConnStateClosed .
	ConnStateClosed
)

// StreamState tags a stream state.
type StreamState uint8

const (
	// StreamStateOpened .
	StreamStateOpened StreamState = iota

	// StreamStateClosed .
	StreamStateClosed
)

// Anchor .
type Anchor interface {
	Ls() Iterator
	Walk(context.Context, []string) Anchor
}

// Iterator .
type Iterator interface {
	Err() error
	Next() bool
	Path() string // subpath
	Anchor() Anchor
}
