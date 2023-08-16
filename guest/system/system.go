package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/stealthrocket/wazergo/types"
)

var conn *rpc.Conn

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		panic(err)
	}

	return T(client), client.Release
}

func Poll() error {
	errno := pollHost()
	if errno != 0 {
		return types.Errno(errno)
	}

	return nil
}
