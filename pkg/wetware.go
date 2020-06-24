package ww

import (
	"context"

	"github.com/libp2p/go-libp2p-core/protocol"
)

const (
	// DefaultNamespace .
	DefaultNamespace = "ww"

	// Protocol id for wetware RPC
	Protocol = protocol.ID("/ww/0.0.0")
)

// Anchor is a node in a cluster-wide, hierarchical namespace.
type Anchor interface {
	String() string
	Path() []string
	Ls(context.Context) ([]Anchor, error)
	Walk(context.Context, []string) Anchor
	// Resolve() (Anchor, error)
}
