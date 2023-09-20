package system

import (
	"context"
	"log/slog"

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
	sock := &Socket{
		Host:  NewPipe(),
		Guest: NewPipe(),
	}

	wazergo.Configure(sock, opts...)
	sock.bind(ctx)
	return
}
