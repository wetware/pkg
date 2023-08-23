package system

import (
	"context"
	"io"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	conn := rpc.NewConn(socket{}, nil)
	runtime.SetFinalizer(conn, func(c io.Closer) error {
		return c.Close()
	})

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return failure[T](err)
	}

	return T(client), func() {
		client.Release()
		conn.Close()
	}
}

func failure[T ~capnp.ClientKind](err error) (T, capnp.ReleaseFunc) {
	return T(capnp.ErrorClient(err)), func() {}
}
