package system

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/log"
)

// module for wetware Host
var module wazergo.HostModule[*Socket] = functions{
	"__sock_read":  wazergo.F2((*Socket).Read),
	"__sock_write": wazergo.F2((*Socket).Write),
	"__sock_close": wazergo.F0((*Socket).close),
}

// Instantiate the system host module.  If instantiation fails, the
// returned context is expired, and the ctx.Err() method returns the
// offending error.
func Instantiate(ctx context.Context, r wazero.Runtime, sess auth.Session) (*wazergo.ModuleInstance[*Socket], context.Context, error) {
	// Instantiate the host module and bind it to the context.
	instance, err := wazergo.Instantiate(ctx, r, module,
		withLogger(slog.Default()),
		withSession(sess))
	if err == nil {
		// Bind the module instance to the context, so that the caller can
		// access it.
		ctx = wazergo.WithModuleInstance(ctx, instance)
	}

	return instance, ctx, err
}

type Option = wazergo.Option[*Socket]

func withLogger(log log.Logger) Option {
	return wazergo.OptionFunc(func(h *Socket) {
		h.Logger = log
	})
}

func withSession(sess auth.Session) Option {
	return wazergo.OptionFunc(func(h *Socket) {
		h.Session = sess
	})
}

// The `functions` type impements `Module[*Module]`, providing the
// module name, map of exported functions, and the ability to create
// instances of the module type
type functions wazergo.Functions[*Socket]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Socket] {
	return (wazergo.Functions[*Socket])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (out *Socket, err error) {
	host, guest := Pipe()

	sock := &Socket{
		Host:  synchronized(ctx, host),
		Guest: guest,
	}

	wazergo.Configure(sock, opts...)
	sock.Bind(ctx)
	return
}

// Socket is a system socket that uses the host's IP stack.
type Socket struct {
	Logger      log.Logger
	Host, Guest io.ReadWriteCloser
	Waiter      interface{ Wait(context.Context) error }
	Session     auth.Session
	conn        *rpc.Conn
}

func (sock *Socket) Close(context.Context) error {
	sock.Session.Logout()

	return sock.conn.Close()
}

func (sock *Socket) close(ctx context.Context) types.Error {
	if err := sock.Close(ctx); err != nil {
		types.Fail(err)
	}

	return types.OK
}

func (sock *Socket) Bind(ctx context.Context) {
	// NOTE:  no auth is actually performed here.  The client doesn't
	// even need to pass a valid signer; the login call always succeeds.
	server := core.Terminal_NewServer(sock.Session)
	client := capnp.NewClient(server)

	options := &rpc.Options{
		ErrorReporter:   ErrorReporter{Logger: sock.Logger},
		BootstrapClient: client,
	}

	sock.conn = rpc.NewConn(rpc.NewStreamTransport(sock.Host), options)
}

// Send is called by the GUEST to send data to the host.
func (sock *Socket) Write(ctx context.Context, b types.Bytes, size types.Pointer[types.Uint32]) types.Errno {
	n, err := io.Copy(sock.Guest, bytes.NewReader(b))
	size.Store(types.Uint32(n))
	return types.AsErrno(err)
}

// Recv is called by the GUEST to receive data to the host.
func (sock *Socket) Read(ctx context.Context, b types.Bytes, size types.Pointer[types.Uint32]) types.Error {
	n, err := sock.Guest.Read(b)
	size.Store(types.Uint32(n))
	if err == nil {
		return types.OK
	}

	return types.Fail(err)
}

type waiter struct {
	Ctx context.Context
	io.ReadWriteCloser
	WaitStrategy interface {
		Wait(context.Context) error
	}
}

func synchronized(ctx context.Context, rwc io.ReadWriteCloser) waiter {
	return waiter{
		Ctx:             ctx,
		ReadWriteCloser: rwc,
		WaitStrategy:    Retry(time.Microsecond * 100),
	}
}

func (s waiter) Read(b []byte) (n int, err error) {
	var x int
	for {
		x, err = s.ReadWriteCloser.Read(b)
		n += x
		b = b[x:]

		if err != ErrUnderflow {
			return
		}

		if err = s.WaitStrategy.Wait(s.Ctx); err != nil {
			return
		}
	}
}

func (s waiter) Write(b []byte) (n int, err error) {
	var x int
	for {
		x, err = s.ReadWriteCloser.Write(b)
		n += x
		b = b[x:]

		if err != ErrOverflow {
			return
		}

		if err = s.WaitStrategy.Wait(s.Ctx); err != nil {
			return
		}
	}
}

type Retry time.Duration

func (r Retry) Wait(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r))
	defer cancel()

	if err := ctx.Err(); err != context.DeadlineExceeded {
		return err
	}

	return nil
}
