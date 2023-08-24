package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	sock := &socket{}
	sock.ctx, sock.cancel = context.WithCancel(ctx)

	conn := rpc.NewConn(sock, nil)

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		defer conn.Close()
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
