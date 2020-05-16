package ww

import (
	"context"

	protocol "github.com/libp2p/go-libp2p-core/protocol"
)

const (
	// DefaultNamespace .
	DefaultNamespace = "ww"

	// ClientUAgent is the user agent for a client connection.
	// TODO:  consider moving to package `client` and unexporting when
	// 		  event.EvtPeerConnectednessChanged becomes available.
	ClientUAgent = "ww-client"

	// Protocol is the common prefix for all wetware wire protocols.
	Protocol = "/ww/0.0.0"

	// ClusterProtocol is the protocol ID for cluster-level operations
	ClusterProtocol = protocol.ID(Protocol + "/cluster")

	// AnchorProtocol is the protocol ID for interacting with remote anchors
	AnchorProtocol = protocol.ID(Protocol + "/anchor")
)

// Anchor .
type Anchor interface {
	Ls(context.Context) Iterator
	Walk(context.Context, []string) (Anchor, error)
}

// Iterator .
type Iterator interface {
	Err() error
	Next() bool
	Path() string // subpath
	Anchor() Anchor
}
