package anchor

import "context"

type Anchor interface {
	Ls(ctx context.Context, path []string) (AnchorIterator, error)
	Walk(ctx context.Context, path []string) (Anchor, error)
}

type AnchorIterator interface {
	Next(context.Context) error
	Finish()
	Anchor() Anchor
}

type Host interface {
	Anchor
	Host() string
}

type Container interface {
	Anchor
	Set(ctx context.Context, data []byte) error
	Get(ctx context.Context) ([]byte, error)
}
