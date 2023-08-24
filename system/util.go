package system

import (
	"context"

	"github.com/tetratelabs/wazero/api"
	"go.uber.org/multierr"
)

type Closer struct {
	Head, Tail api.Closer
}

func (c *Closer) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}

	return multierr.Append(
		c.Head.Close(ctx),
		c.Tail.Close(ctx))
}

func (c *Closer) WithCloser(closer api.Closer) *Closer {
	if closer == nil {
		return c
	}

	return &Closer{
		Head: closer,
		Tail: c,
	}
}
