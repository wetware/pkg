package ww

import "context"

const (
	// DefaultNamespace .
	DefaultNamespace = "ww"

	// ClientUAgent is the user agent for a client connection.
	// TODO:  consider moving to package `client` and unexporting when
	// 		  event.EvtPeerConnectednessChanged becomes available.
	ClientUAgent = "ww-client"
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
