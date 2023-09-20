package system

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/jpillora/backoff"
	"go.uber.org/multierr"

	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/log"
)

type Closer struct {
	api.Closer
	Next *Closer
}

func (c Closer) Close(ctx context.Context) (err error) {
	for c, next := c.Closer, c.Next; c != nil; c, next = next.Closer, next.Next {
		err = multierr.Append(err, c.Close(ctx))
	}

	return err
}

type Module interface {
	Instantiate(ctx context.Context, r wazero.Runtime, sess auth.Session) (api.Closer, context.Context, error)
}

// module for wetware Host
var module wazergo.HostModule[*Socket] = functions{
	// TODO(soon):  socket exports
	"__sock_close": wazergo.F0((*Socket).close),
	"__sock_send":  wazergo.F1((*Socket).Send),
	"__sock_recv":  wazergo.F1((*Socket).Recv),
	// "foo": ,
	// "bar": F1((*NetSock).Bar),
}

// Instantiate the system host module.  If instantiation fails, the
// returned context is expired, and the ctx.Err() method returns the
// offending error.
func Instantiate(ctx context.Context, r wazero.Runtime, sess auth.Session) (*wazergo.ModuleInstance[*Socket], context.Context, error) {
	// l, err := net.Listen("tcp", ":0") // TODO:  localhost?
	// if err != nil {
	// 	return nil, ctx, fmt.Errorf("net: listen: %w", err)
	// }
	// defer l.Close()

	// addr := l.Addr().(*net.TCPAddr)
	host, guest := net.Pipe()
	ctx = context.WithValue(ctx, keyHostPipe{}, host)

	// Instantiate the host module and bind it to the context.
	instance, err := wazergo.Instantiate(ctx, r, module,
		withLogger(slog.Default()),
		withNetConn(guest),
		withSession(sess))
	if err == nil {
		// Bind the module instance to the context, so that the caller can
		// access it.
		ctx = wazergo.WithModuleInstance(ctx, instance)

		// // The system socket enables the creation of non-blocking TCP conns
		// // inside of the WASM module.  The host will pre-open the TCP port
		// // and pass it to the guest through a file descriptor.
		// ctx = sock.WithConfig(ctx, sock.NewConfig().
		// 	WithTCPListener("", addr.Port))
	}

	return instance, ctx, err

}

type Option = wazergo.Option[*Socket]

func withLogger(log log.Logger) Option {
	return wazergo.OptionFunc(func(h *Socket) {
		h.Logger = log
	})
}

func withNetConn(conn net.Conn) Option {
	return wazergo.OptionFunc(func(h *Socket) {
		h.Conn = conn
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
	wazergo.Configure(new(Socket), append(opts, wazergo.OptionFunc(func(h *Socket) {
		var b = backoff.Backoff{
			Min:    time.Millisecond * 1,
			Max:    time.Minute,
			Jitter: true,
		}

		// retry in a loop until context is canceled; back-off exponentially.
		for err = h.dial(ctx); err != nil; err = h.dial(ctx) {
			h.Logger.Debug("failed to dial host socket",
				"error", err,
				"attempt", b.Attempt(),
				"backoff", b.ForAttempt(b.Attempt()))

			select {
			case <-time.After(b.Duration()):
				h.Logger.Debug("resuming",
					"attempt", b.Attempt())

			case <-ctx.Done():
				err = ctx.Err()
			}
		}

		out = h // pass up the call stack
	}))...)

	return
}

type keyHostPipe struct{}
