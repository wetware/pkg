package system

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/jpillora/backoff"
	"golang.org/x/exp/slog"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
)

// file descriptor for first pre-openned file descriptor.
const PREOPENED_FD = 3

type Dialer interface {
	Dial(context.Context, net.Addr) (net.Conn, error)
}

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	f := load()

	conn, err := connect(ctx, f, fileDialer{})
	if err != nil {
		defer f.Close()
		panic(err)
	}

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		defer f.Close()
		panic(err)
	}

	return T(client), func() {
		defer f.Close()
		defer conn.Close()
		defer client.Release()
	}
}

func connect(ctx context.Context, addr net.Addr, d Dialer) (_ *rpc.Conn, err error) {
	opt := rpc.Options{
		// TODO:  ErrorReporter{Logger: slog.Default()}
	}

	b := backoff.Backoff{
		Jitter: true,
		Min:    time.Microsecond * 100,
		Max:    time.Millisecond * 100,
	}

	var conn net.Conn
	for {
		conn, err = d.Dial(ctx, addr)
		if err == nil {
			break
		}

		slog.Debug("dial failed",
			"addr", addr,
			"error", err,
			"attempt", b.Attempt(),
			"backoff", b.ForAttempt(b.Attempt()))

		select {
		case <-time.After(b.Duration()):
		case <-ctx.Done():
			err = ctx.Err()
			return
		}

	}

	return upgrade(conn, &opt), nil
}

func upgrade(pipe io.ReadWriteCloser, opt *rpc.Options) *rpc.Conn {
	return rpc.NewConn(stream(pipe), opt)
}
