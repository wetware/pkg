package cluster

import (
	"context"

	"github.com/wetware/ww/pkg/vat"
)

var (
	AnchorCapability = vat.BasicCap{
		"anchor/packed",
		"anchor"}
)

type Anchor interface {
	Path() []string
	Name() string
	Ls(ctx context.Context) (AnchorIterator, error)
	Walk(ctx context.Context, path []string) (Anchor, error)
}

type AnchorIterator interface {
	Next(context.Context) bool
	Finish()
	Anchor() Anchor
	Err() error
}

type Container interface {
	Set(ctx context.Context, data []byte) error
	Get(ctx context.Context) (data []byte, release func())
}
