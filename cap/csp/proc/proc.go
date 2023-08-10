// Package proc provides the WebAssembly host module for Wetware processes
package proc

import (
	"context"
	"io"
	"net"

	"golang.org/x/exp/slog"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exp/bufferpool"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/stealthrocket/wazergo"
	t "github.com/stealthrocket/wazergo/types"
	wasm "github.com/tetratelabs/wazero"
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

var fs wazergo.HostModule[*Module] = functions{
	"__host_write": wazergo.F2((*Module).write),
	"__host_read":  wazergo.F2((*Module).read),
	"__host_close": wazergo.F0((*Module).close),
}

// BindModule instantiates a host module instance and binds it to the supplied
// runtime.  The returned module instance is bound to the lifetime of r.
func BindModule(ctx context.Context, r wasm.Runtime, opt ...Option) *wazergo.ModuleInstance[*Module] {
	// We use BindModule to avoid exporting fs as a public variable, since this
	// would allow users to mutate it.
	return wazergo.MustInstantiate(ctx, r, fs, opt...)
}

type functions wazergo.Functions[*Module]

func (functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Module] {
	return (wazergo.Functions[*Module])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Module, error) {
	module := &Module{}
	wazergo.Configure(module, opts...)

	var rwc io.ReadWriteCloser
	rwc, module.pipe = net.Pipe()

	module.conn = rpc.NewConn(rpc.NewStreamTransport(rwc), &rpc.Options{
		BootstrapClient: module.bootstrap,
		ErrorReporter:   module.errReporter(),
	})

	return module, nil
}

type Option = wazergo.Option[*Module]

// WithClient sets the bootstrap client provided to the guest code.
func WithClient[Client ~capnp.ClientKind](c Client) Option {
	return wazergo.OptionFunc(func(m *Module) {
		m.bootstrap = capnp.Client(c)
	})
}

// WithLogger sets the error logger for the capnp transport.   If
// l == nil, logging is disabled.
func WithLogger(l Logger) Option {
	if l == nil {
		l = slog.Default()
	}

	return wazergo.OptionFunc(func(m *Module) {
		m.logger = l
	})
}

type Module struct {
	pipe io.ReadWriteCloser
	conn io.Closer

	logger    Logger
	bootstrap capnp.Client
}

func (m Module) Close(context.Context) error {
	return m.conn.Close() // close the host side of the connection
}

func (m Module) write(ctx context.Context, b t.Bytes, n t.Pointer[t.Uint32]) t.Error {
	// b is only valid until Write returns.
	//
	// TODO(perf):  can the guest somehow "pin" b to a global map, and
	//              expect the transport to call free(b)?  This would
	//              give us a truly zero-copy transport, though we would
	//              likely have to operate at the Arena level for this.
	p := bufferpool.Default.Get(len(b))
	copy(p, b)

	// Due to a bug in wazergo, we need to use Pointer[Uint32] to extract
	// the number of bytes written.
	//
	// TODO:  revert to Optional[Uint32] when possible.
	u, err := m.pipe.Write(p)
	n.Store(t.Uint32(u))

	if err != nil {
		return t.Err[t.None](err)
	}

	return t.OK
}

func (m Module) read(ctx context.Context, b t.Bytes, n t.Pointer[t.Uint32]) t.Error {
	u, err := m.pipe.Read(b)
	n.Store(t.Uint32(u)) // See Write()

	if err != nil {
		return t.Err[t.None](err)
	}

	return t.OK
}

func (m Module) close(ctx context.Context) t.Error {
	return t.Err[t.None](m.pipe.Close())
}

func (m Module) errReporter() errReporter {
	return errReporter{
		Logger: m.logger,
	}
}

type errReporter struct {
	Logger
}

func (er errReporter) ReportError(err error) {
	er.Logger.Error(err.Error())
}
