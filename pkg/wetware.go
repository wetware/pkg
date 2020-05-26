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

	// Protocol id for wetware RPC
	Protocol = protocol.ID("/ww/0.0.0")
)

// Anchor .
type Anchor interface {
	Ls(context.Context) (Iterator, error)
	Walk(context.Context, []string) (Anchor, error)
}

// Iterator .
type Iterator interface {
	Err() error
	Next() bool
	Path() string // subpath
	Anchor() Anchor
}
