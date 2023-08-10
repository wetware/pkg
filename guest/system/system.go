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
	"golang.org/x/exp/slog"
)

const (
	// file descriptor for pre-openned TCP socket
	PREOPENED_FD = 3
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
}

// Boot bootstraps and resolves the Capnp client attached
// to the other end of the pre-openned TCP connection.
// capnp.Client will be capnp.ErrorClient if an error ocurred.
func Boot(ctx context.Context) (capnp.Client, capnp.ReleaseFunc) {
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
		return capnp.ErrorClient(err), func() {}
	}
	closers = append(closers, l)

	tcpConn, err := l.Accept()
	if err != nil {
		defer release()
		return capnp.ErrorClient(err), func() {}
	}
	closers = append(closers, tcpConn)

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	closers = append(closers, conn)

	client := conn.Bootstrap(ctx)

	err = client.Resolve(ctx)
	if err != nil {
		defer release()
		return capnp.ErrorClient(err), func() {}
	}

	return client, release
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

// errLogger panics when an error occurs
type errLogger struct {
	Logger
}

func (e errLogger) ReportError(err error) {
	if err != nil {
		if e.Logger == nil {
			e.Logger = slog.Default()
		}

		e.Debug("rpc: connection closed",
			"error", err)
	}
}
