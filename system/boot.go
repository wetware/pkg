package system

import (
	"context"

	local "github.com/libp2p/go-libp2p/core/host"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type Dialer interface {
	DialRPC(context.Context, local.Host) (*rpc.Conn, error)
}

func Bootstrap[T ~capnp.ClientKind](ctx context.Context, h local.Host, d Dialer) (T, error) {
	conn, err := d.DialRPC(ctx, h)
	if err != nil {
		return T{}, err
	}

	client := conn.Bootstrap(ctx)
	return T(client), client.Resolve(ctx)
}
