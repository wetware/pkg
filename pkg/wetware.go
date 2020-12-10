//go:generate mockgen -package mock_ww -destination ../internal/test/mock/pkg/mock_wetware.go github.com/wetware/ww/pkg Logger,Loggable,Any,Anchor

// Package ww contains core interfaces and symbols
package ww

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/lthibault/log"
	"github.com/wetware/ww/internal/mem"
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

// Logger is used throughout the Wetware codebase to provide
// observability.
//
// See options for inidivdual packages to customize logging.
type Logger interface{ log.Logger }

// Loggable representation of an arbitrary type.
type Loggable interface{ log.Loggable }

// Any is a generic value type
type Any interface {
	Value() mem.Any
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
