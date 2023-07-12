package ww

/*
 * The contents of this file will be moved to the ww repository
 */

import (
	"context"
	"io"
	"net"
	"os"
	"syscall"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	csp "github.com/wetware/ww/pkg/csp"
)

// Default file descriptor for Wazero pre-openned TCP connections
const (
	// file descriptor for pre-openned TCP socket
	PREOPENED_FD = 3

	// Inbox in which each element will be found by default on the inbox
	HOST_INDEX = 0
	ARGS_INDEX = 1
)

// closer contains a slice of Closers that will be closed when this type itself is closed
type closer struct {
	closers []io.Closer
}

func (c closer) Close() error {
	for _, closer := range c.closers {
		defer closer.Close()
	}
	return nil
}

// add a new closer to the list
func (c closer) add(closer io.Closer) {
	c.closers = append(c.closers, closer)
}

// return the a TCP listener from pre-opened tcp connection by using the fd
func preopenedListener(c closer) (net.Listener, error) {
	f := os.NewFile(uintptr(PREOPENED_FD), "")

	if err := syscall.SetNonblock(PREOPENED_FD, false); err != nil {
		return nil, err
	}

	c.add(f)

	l, err := net.FileListener(f)
	if err != nil {
		return nil, err
	}
	c.add(l)

	return l, err
}

// BootstrapClient bootstraps and resolves the Capnp client attached
// to the other end of the pre-openned TCP connection
func BootstrapClient(ctx context.Context) (capnp.Client, io.Closer, error) {
	closer := closer{
		closers: make([]io.Closer, 0),
	}

	l, err := preopenedListener(closer)
	if err != nil {
		return capnp.Client{}, closer, err
	}

	tcpConn, err := l.Accept()
	if err != nil {
		return capnp.Client{}, closer, err
	}

	closer.add(tcpConn)

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		ErrorReporter: errLogger{},
	})
	closer.add(conn)

	client := conn.Bootstrap(ctx)

	err = client.Resolve(ctx)

	return client, closer, err
}

// Init should be called whenever a process starts. It loads and resolves
// any capabilities left by call that created the process.
func Init(ctx context.Context) ([]capnp.Client, io.Closer, error) {
	inbox, closer, err := BootstrapClient(ctx)
	if err != nil {
		return nil, closer, err
	}

	if err := inbox.Resolve(context.Background()); err != nil {
		return nil, closer, err
	}

	clients, err := csp.Inbox(inbox).Open(context.TODO())

	return clients, closer, err
}

// errLogger panics when an error occurs
type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		panic(err)
	}
}
