package system

import (
	"context"
	"io"
	"runtime"

	local "github.com/libp2p/go-libp2p/core/host"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type Dialer interface {
	DialRPC(context.Context, local.Host) (*rpc.Conn, error)
}

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) T {
	conn, err := FDSockDialer{}.DialRPC(ctx)
	if err != nil {
		return failure[T](err)
	}
	runtime.SetFinalizer(conn, func(c io.Closer) error {
		return c.Close()
	})

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return failure[T](err)
	}

	return T(client)
}

func failure[T ~capnp.ClientKind](err error) T {
	return T(capnp.ErrorClient(err))
}
