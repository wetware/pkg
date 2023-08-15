package system

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/jpillora/backoff"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/wetware/pkg/util/log"
	"go.uber.org/multierr"
	"golang.org/x/exp/slog"
)

type Closer struct {
	api.Closer
	Next api.Closer
}

func (c Closer) Close(ctx context.Context) (err error) {
	slog.Warn("***",
		"closer", c.Closer,
		"next", c.Next)

	return multierr.Append(
		c.Closer.Close(ctx),
		c.Next.Close(ctx))
}

func Instantiate[T ~capnp.ClientKind](ctx context.Context, r wazero.Runtime, t T) (api.Closer, context.Context, error) {
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return nil, ctx, err
	}

	// instantiate wetware
	l, err := net.Listen("tcp", ":0") // TODO:  localhost?
	if err != nil {
		defer c.Close(ctx)
		return nil, ctx, fmt.Errorf("net: listen: %w", err)
	}
	l.Close()

	// Instantiate the host module and bind it to the context.
	addr := l.Addr()
	instance, err := wazergo.Instantiate(ctx, r, module,
		logger(slog.Default()),
		address(addr),
		bootstrap(t))
	if err == nil {
		// Bind the module instance to the context, so that the caller can
		// access it.
		ctx = wazergo.WithModuleInstance(ctx, instance)

		// The system socket enables the creation of non-blocking TCP conns
		// inside of the WASM module.  The host will pre-open the TCP port
		// and pass it to the guest through a file descriptor.
		ctx = sock.WithConfig(ctx, sock.NewConfig().
			WithTCPListener("", addr.(*net.TCPAddr).Port))
	}

	return Closer{Closer: c, Next: instance}, ctx, err
}

// module for wetware Host
var module wazergo.HostModule[*NetSock] = functions{
	// TODO(soon):  socket exports
	// "foo": F0((*NetSock).Foo),
	// "bar": F1((*NetSock).Bar),
}

type Option = wazergo.Option[*NetSock]

func address(addr net.Addr) Option {
	return wazergo.OptionFunc(func(h *NetSock) {
		h.Addr = addr
	})
}

func logger(log log.Logger) Option {
	return wazergo.OptionFunc(func(h *NetSock) {
		h.Opt.ErrorReporter = ErrorReporter{
			Logger: log,
		}
	})
}

func bootstrap[T ~capnp.ClientKind](t T) Option {
	return wazergo.OptionFunc(func(h *NetSock) {
		h.Opt.BootstrapClient = capnp.Client(t)
	})
}

// The `functions` type impements `Module[*Module]`, providing the
// module name, map of exported functions, and the ability to create
// instances of the module type
type functions wazergo.Functions[*NetSock]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*NetSock] {
	return (wazergo.Functions[*NetSock])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (out *NetSock, err error) {
	wazergo.Configure(new(NetSock), append(opts, wazergo.OptionFunc(func(sock *NetSock) {
		var b = backoff.Backoff{
			Min:    time.Millisecond * 1,
			Max:    time.Minute,
			Jitter: true,
		}

		// retry in a loop until context is canceled; back-off exponentially.
		for err = sock.dial(ctx); err != nil; err = sock.dial(ctx) {
			if errors.Is(err, context.Canceled) {
				err = context.Canceled
				return
			} else if errors.Is(err, context.DeadlineExceeded) {
				return
			}

			slog.Warn("failed to dial host socket",
				"error", err,
				"attempt", b.Attempt(),
				"backoff", b.ForAttempt(b.Attempt()))

			select {
			case <-time.After(b.Duration()):
				slog.Warn("resuming",
					"attempt", b.Attempt())

			case <-ctx.Done():
				err = ctx.Err()
			}
		}

		out = sock // pass up the call stack
	}))...)

	return
}
