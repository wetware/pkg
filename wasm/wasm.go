package ww

/*
 * The contents of this file will be moved to the ww repository
 */

import (
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"syscall"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	csp "github.com/wetware/ww/pkg/csp"
)

// Default file descriptor for Wazero pre-openned TCP connections
const (
	// file descriptor for pre-openned TCP socket
	PREOPENED_FD = 3

	// BootContext in which each element will be found by default on the bootContext
	SELF_INDEX       = 0
	ARGS_INDEX       = 1
	CAPS_START_INDEX = 2

	// Argument order
	ARG_PID = 0 // PID of the process
	ARG_MD5 = 1 // md5 sum of the process, used to self-replicate
)

// Self contains the info a WASM process will need for:
type Self struct {
	Args    []string       // Receiving parameters.
	Caps    []capnp.Client // Communication.
	Closers io.Closer      // Cleaning up.
	Md5Sum  []byte         // Self-replicating.
	Pid     uint32         // Indetifying self.
}

func (s *Self) Close() error {
	return s.Closers.Close()
}

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

// OpenBootContext may be called whenever a process starts. It loads and resolves
// any capabilities left by call that created the process. Not required if
// Init was called.
func OpenBootContext(ctx context.Context) ([]capnp.Client, io.Closer, error) {
	bootContext, closer, err := BootstrapClient(ctx)
	if err != nil {
		return nil, closer, err
	}

	if err := bootContext.Resolve(context.Background()); err != nil {
		return nil, closer, err
	}

	clients, err := csp.BootContext(bootContext).Open(context.TODO())

	return clients, closer, err
}

func Init(ctx context.Context) (Self, error) {
	clients, closers, err := OpenBootContext(ctx)
	if err != nil {
		return Self{}, err
	}
	selfArgs, err := csp.Args(clients[SELF_INDEX]).Args(ctx)
	if err != nil {
		return Self{}, err
	}
	pid64, err := strconv.ParseUint(selfArgs[ARG_PID], 10, 32)
	if err != nil {
		return Self{}, err
	}
	md5sum := selfArgs[ARG_MD5]

	args, err := csp.Args(clients[ARGS_INDEX]).Args(ctx)
	if err != nil {
		return Self{}, err
	}

	return Self{
		Args:    args,
		Caps:    clients[CAPS_START_INDEX:],
		Closers: closers,
		Md5Sum:  []byte(md5sum),
		Pid:     uint32(pid64),
	}, nil
}

// errLogger panics when an error occurs
type errLogger struct{}

func (e errLogger) ReportError(err error) {
	if err != nil {
		panic(err)
	}
}
