package system

import (
	"context"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/pkg/system"
	"golang.org/x/exp/slog"
)

const (
	// file descriptor for first pre-openned file descriptor.
	PREOPENED_FD = 3
)

// FDSockDialer binds to a pre-opened file descriptor (usually a TCP socket),
// and provides an *rcp.Conn to the host.
type FDSockDialer struct{}

func (s FDSockDialer) DialRPC(context.Context) (*rpc.Conn, error) {
	f := os.NewFile(uintptr(PREOPENED_FD), "")
	if err := syscall.SetNonblock(PREOPENED_FD, false); err != nil {
		return nil, err
	}

	// Make sure we eventually release the file descriptor.
	runtime.SetFinalizer(f, func(c io.Closer) error {
		return c.Close()
	})

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	raw, err := l.Accept()
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(rpc.NewStreamTransport(raw), &rpc.Options{
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default().WithGroup("guest"),
		},
	})

	return conn, nil
}
