package system

import (
	"context"
	"fmt"
	"net"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/jpillora/backoff"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/experimental/sock"
	"github.com/wetware/pkg/util/log"
	"golang.org/x/exp/slog"
)

// Module for wetware Host
var Module wazergo.HostModule[*Host] = functions{
	// "answer": F0((*Module).Answer),
	// "double": F1((*Module).Double),
}

// Instantiate the system host module.  If instantiation fails, the
// returned context is expired, and the ctx.Err() method returns the
// offending error.
func Instantiate[T ~capnp.ClientKind](ctx context.Context, r wazero.Runtime, t T) (*wazergo.ModuleInstance[*Host], context.Context) {
	l, err := net.Listen("tcp", ":0") // TODO:  localhost?
	if err != nil {
		return nil, failuref("net: listen: %w", err)
	}
	defer l.Close()

	addr := l.Addr().(*net.TCPAddr)

	// Instantiate the host module and bind it to the context.
	instance, err := wazergo.Instantiate(ctx, r, Module,
		logger(slog.Default()),
		transport(addr),
		bootstrap(t))
	if err != nil {
		return nil, failure(err)
	}
	ctx = wazergo.WithModuleInstance(ctx, instance)

	// The system socket enables the creation of non-blocking TCP conns
	// inside of the WASM module.  The host will pre-open the TCP port
	// and pass it to the guest through a file descriptor.
	ctx = sock.WithConfig(ctx, sock.NewConfig().
		WithTCPListener("", addr.Port))

	return instance, ctx
}

func failuref(format string, args ...any) context.Context {
	return failure(fmt.Errorf(format, args...))
}

func failure(err error) context.Context {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(err)

	return ctx
}

type Option = wazergo.Option[*Host]

func logger(log log.Logger) Option {
	return wazergo.OptionFunc(func(h *Host) {
		h.Logger = log
	})
}

func transport(addr *net.TCPAddr) Option {
	return wazergo.OptionFunc(func(h *Host) {
		h.TCPAddr = addr
	})
}

func bootstrap[T ~capnp.ClientKind](t T) Option {
	return wazergo.OptionFunc(func(h *Host) {
		h.BootstrapClient = capnp.Client(t)
	})
}

// The `functions` type impements `Module[*Module]`, providing the
// module name, map of exported functions, and the ability to create
// instances of the module type
type functions wazergo.Functions[*Host]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Host] {
	return (wazergo.Functions[*Host])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...wazergo.Option[*Host]) (*Host, error) {
	mod := &Host{}
	wazergo.Configure(mod, opts...)

	return mod, mod.dialSock(ctx)
}

type Host struct {
	*net.TCPAddr
	Logger          log.Logger
	BootstrapClient capnp.Client

	conn *rpc.Conn
}

func (h *Host) Close(context.Context) error {
	h.BootstrapClient.Release()

	return h.conn.Close()
}

func (h *Host) dialSock(ctx context.Context) (err error) {
	var b = backoff.Backoff{
		Min: time.Millisecond * 1,
		Max: time.Millisecond * 100,
	}

	for err = h.dialOnce(ctx); err != nil; err = h.dialOnce(ctx) {
		h.Logger.Debug("failed to dial host socket",
			"error", err,
			"attempt", b.Attempt(),
			"backoff", b.ForAttempt(b.Attempt()))

		select {
		case <-time.After(b.Duration()):
			h.Logger.Debug("resuming",
				"attempt", b.Attempt())

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return
}

func (h *Host) dialOnce(ctx context.Context) error {
	var (
		d    net.Dialer
		addr = h.TCPAddr.String()
	)

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		h.conn = rpc.NewConn(rpc.NewStreamTransport(conn), h.options())
	}

	return err
}

func (h *Host) options() *rpc.Options {
	return &rpc.Options{
		ErrorReporter:   log.ErrorReporter{Logger: h.Logger},
		BootstrapClient: h.BootstrapClient,
	}
}
