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

type (
	// Anchor is a node in a cluster-wide, hierarchical namespace.
	Anchor interface {
		String() string
		Path() []string
		Ls(context.Context) ([]Anchor, error)
		Walk(context.Context, []string) Anchor
		// Resolve() (Anchor, error)
	}
)
