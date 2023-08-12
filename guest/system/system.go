package system

/*
 * The contents of this file will be moved to the ww repository
 */

import (
	"context"
	"io"
	"net"
	"os"
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
func Boot[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	var closers []io.Closer
	release := func() {
		for i := range closers {
			// call in reverse order, similar to defer
			_ = closers[len(closers)-i-1].Close()
		}
	}

	l, err := preopenedListener(&closers)
	if err != nil {
		defer release()
		return failure[T](err)
	}
	closers = append(closers, l)

	tcpConn, err := l.Accept()
	if err != nil {
		defer release()
		return failure[T](err)
	}
	closers = append(closers, tcpConn)

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: log.ErrorReporter{
			Logger: slog.Default().WithGroup("guest"),
		},
	})
	closers = append(closers, conn)

	client := conn.Bootstrap(ctx)
	return T(client), release
}

func failure[T ~capnp.ClientKind](err error) (T, capnp.ReleaseFunc) {
	return T(capnp.ErrorClient(err)), func() {}
}

// return the a TCP listener from pre-opened tcp connection by using the fd
func preopenedListener(closers *[]io.Closer) (net.Listener, error) {
	f := os.NewFile(uintptr(PREOPENED_FD), "")

	if err := syscall.SetNonblock(PREOPENED_FD, false); err != nil {
		return nil, err
	}
	*closers = append(*closers, f)

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	*closers = append(*closers, l)

	return l, err
}
