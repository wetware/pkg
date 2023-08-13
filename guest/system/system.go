package system

/*
 * The contents of this file will be moved to the ww repository
 */

import (
	"context"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/pkg/util/log"
	"golang.org/x/exp/slog"
)

const (
	// file descriptor for pre-openned TCP socket
	PREOPENED_FD = 3
)

// Boot bootstraps and resolves the Capnp client attached
// to the other end of the pre-openned TCP connection.
// capnp.Client will be capnp.ErrorClient if an error ocurred.
func Boot[T ~capnp.ClientKind](ctx context.Context) (T, error) {
	sock, err := socket(ctx)
	if err != nil {
		return T{}, err
	}

	conn := rpc.NewConn(rpc.NewStreamTransport(sock), &rpc.Options{
		ErrorReporter: &log.ErrorReporter{
			Logger: slog.Default(),
		},
	})
	go func() {
		defer conn.Close()

		select {
		case <-ctx.Done():
		case <-conn.Done():
		}
	}()

	client := conn.Bootstrap(ctx)
	return T(client), client.Resolve(ctx)
}

func socket(ctx context.Context) (net.Conn, error) {
	f := os.NewFile(uintptr(PREOPENED_FD), "")
	runtime.SetFinalizer(f, func(c io.Closer) error {
		return c.Close()
	})

	if err := syscall.SetNonblock(PREOPENED_FD, false); err != nil {
		return nil, err
	}

	l, err := net.FileListener(f)
	if err != nil {
		defer f.Close()
		return nil, err
	}
	defer l.Close()

	return l.Accept()
}
