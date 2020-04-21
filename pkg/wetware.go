package ww

import (
	"context"
)

const (
	// ClientUAgent is the user agent for a client connection.
	ClientUAgent = "ww-client"

// 	// ConnTypeClient indicates a client connection
// 	ConnTypeClient = iota

// 	// ConnTypeServer inidicates a server connection
// 	ConnTypeServer
)

// type (
// 	// EvtConnectionEstablished fires when a connection has been successfully negotiated,
// 	// and is in a usable state.
// 	EvtConnectionEstablished struct {
// 		ID   peer.ID
// 		Type ConnType
// 	}
// )

// // ConnType tags a connection as belonging to a client or server.
// type ConnType uint8

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
