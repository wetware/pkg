package ww

import (
	"context"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/pkg/errors"
	"github.com/wetware/ww/pkg/mem"
)

const (
	// DefaultNamespace .
	DefaultNamespace = "ww"

	// Protocol is the base protocol id for wetware RPC.
	Protocol = protocol.ID("/ww/0.0.0")

	// AnchorProtocol id for Anchor RPC.
	AnchorProtocol = Protocol + "/anchor"

	// LanguageProtocol for Language RPC.
	LanguageProtocol = Protocol + "/lang"
)

var (
	// ErrAnchorNotEmpty is returned by Anchor.Store when the
	// anchor contains a value.
	ErrAnchorNotEmpty = errors.New("anchor contains value")
)

// Any is a generic value type
type Any interface {
	SExpr() (string, error)
	MemVal() mem.Value
}

// Anchor is a node in a cluster-wide, hierarchical namespace.
type Anchor interface {
	Name() string
	Path() []string
	Ls(context.Context) ([]Anchor, error)
	Walk(context.Context, []string) Anchor
	Load(context.Context) (Any, error)
	Store(context.Context, Any) error
	Go(context.Context, ...Any) (Any, error)
	// Resolve() (Anchor, error)
}
