package view

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/exp/slog"
)

type HostModule[T ~capnp.ClientKind] struct{}

func (HostModule[T]) Instantiate(ctx context.Context, r wazero.Runtime, t T) (api.Closer, context.Context, error) {
	slog.Default().Warn("stub call to view.HostModule.Instantiate:: TODO")
	return nil, ctx, nil
}
