package system

import (
	"context"
	"errors"
	"io"
	"runtime"

	local "github.com/libp2p/go-libp2p/core/host"

	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/csp"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

type Dialer interface {
	DialRPC(context.Context, local.Host) (*rpc.Conn, error)
}

func Bootstrap(ctx context.Context) ([]capnp.Client, []capnp.ReleaseFunc, error) {
	conn, err := FDSockDialer{}.DialRPC(ctx)
	if err != nil {
		return nil, nil, err
	}

	runtime.SetFinalizer(conn, func(c io.Closer) error {
		return c.Close()
	})

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return nil, nil, err
	}

	return bootstrapAll(ctx, csp.ProcessBootstrap(client))
}

func bootstrapAll(ctx context.Context, bootstrap csp.ProcessBootstrap) ([]capnp.Client, []capnp.ReleaseFunc, error) {
	caps := make([]capnp.Client, 0)
	releases := make([]capnp.ReleaseFunc, 0)
	for {
		cap, release, hasNext, err := bootstrap.Get(ctx)
		if err != nil {
			return nil, nil, err
		}

		if !hasNext {
			break
		}

		caps = append(caps, cap.AddRef())
		releases = append(releases, release)
	}

	return caps, releases, nil
}

func Login(ctx context.Context, session capnp.Client) (auth.Session, error) {
	sess := capnp.Client(session).AddRef()
	if err := sess.Resolve(ctx); err != nil {
		return auth.Session{}, errors.New("the capability passed to login is not a session")
	}

	term := api.Terminal(sess)

	f, release := term.Login(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	s, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(s).AddRef(), nil
}
