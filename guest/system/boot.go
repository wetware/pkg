package system

import (
	"context"
	"io"
	"runtime"

	local "github.com/libp2p/go-libp2p/core/host"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/auth"

	"capnproto.org/go/capnp/v3/rpc"
)

type Dialer interface {
	DialRPC(context.Context, local.Host) (*rpc.Conn, error)
}

func Bootstrap(ctx context.Context) (auth.Session, error) {
	conn, err := FDSockDialer{}.DialRPC(ctx)
	if err != nil {
		return auth.Session{}, err
	}
	runtime.SetFinalizer(conn, func(c io.Closer) error {
		return c.Close()
	})

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Session{}, err
	}
	term := api.Terminal(client)

	f, release := term.Login(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	sess, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(sess).AddRef(), nil
}
