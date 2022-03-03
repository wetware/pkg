package cluster

import (
	"context"
	"errors"
)

var (
	ErrInvalidPath = errors.New("invalid path")
)

type Anchor interface {
	Path() []string
	Ls(ctx context.Context) (AnchorIterator, error)
	Walk(ctx context.Context, path []string) (Anchor, error)
}

type AnchorIterator interface {
	Next(context.Context) bool
	Finish()
	Anchor() Anchor
	Err() error
}

type Host interface {
	Anchor
	Host() string
}

type Container interface {
	Anchor
	Set(ctx context.Context, data []byte) error
	Get(ctx context.Context) (data []byte, release func())
}
