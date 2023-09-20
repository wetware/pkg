package system

import (
	"context"
	"errors"
	"io"
	"runtime"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/stealthrocket/wazergo/types"
	"golang.org/x/exp/slog"

	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"

	"capnproto.org/go/capnp/v3/rpc"
)

type Dialer interface {
	DialRPC(context.Context, local.Host) (*rpc.Conn, error)
}

func Bootstrap(ctx context.Context) (auth.Session, error) {
	// conn, err := FDSockDialer{}.DialRPC(ctx)
	// if err != nil {
	// 	return auth.Session{}, err
	// }
	// runtime.SetFinalizer(conn, func(c io.Closer) error {
	// 	return c.Close()
	// })
	conn := rpc.NewConn(rpc.NewStreamTransport(socket{ctx}), nil)
	runtime.SetFinalizer(conn, func(c io.Closer) error {
		defer slog.Debug("called finalizer")
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

type socket struct{ context.Context }

func (socket) Read(b []byte) (int, error) {
	return 0, errors.New("NOT IMPLEMENTED")
}

func (socket) Write(b []byte) (int, error) {
	return 0, errors.New("NOT IMPLEMENTED")
}

func (socket) Close() error {
	if errno := sockClose(); errno != 0 {
		return types.Errno(errno)
	}

	return nil
}
