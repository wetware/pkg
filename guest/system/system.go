package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

var conn *rpc.Conn

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		panic(err)
	}

	return T(client), client.Release
}

func Poll() int32 {
	status := pollHost()
	return status
}
