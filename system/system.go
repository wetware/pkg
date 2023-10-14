package system

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/tetratelabs/wazero"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/wetware/pkg/util/log"
)

var SocketModule wazergo.HostModule[*Socket] = functions{
	"_sysread":  wazergo.F2((*Socket).Read),
	"_syswrite": wazergo.F2((*Socket).Write),
	"_sysclose": wazergo.F0((*Socket).close),
}

type SocketBinder interface {
	BindSocket(ctx context.Context) io.ReadWriteCloser
}

func Instantiate(ctx context.Context, r wazero.Runtime, b SocketBinder) (*wazergo.ModuleInstance[*Socket], error) {
	return wazergo.Instantiate(ctx, r, SocketModule, Bind(ctx, b))
}

type Option = wazergo.Option[*Socket]

func WithLogger(log log.Logger) Option {
	return wazergo.OptionFunc(func(sock *Socket) {
		sock.Logger = log
	})
}

func Bind(ctx context.Context, p SocketBinder) Option {
	return wazergo.OptionFunc(func(sock *Socket) {
		sock.Pipe = p.BindSocket(ctx)
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

func (f functions) Instantiate(ctx context.Context, opts ...Option) (sock *Socket, err error) {
	sock = &Socket{}
	wazergo.Configure(sock, opts...)
	return
}

// Socket is a system socket that uses the host's IP stack.
type Socket struct {
	Logger log.Logger
	Pipe   io.ReadWriteCloser
}

func (sock *Socket) Shutdown() {
	if err := sock.Pipe.Close(); err != nil {
		// Our in-process net.Conn MUST successfully close.   This allows us to
		// guarantee that no access to the object exists after Shutdown returns.
		panic(err)
	}
}

func (sock *Socket) Close(context.Context) error {
	return sock.Pipe.Close()
}

func (sock *Socket) close(ctx context.Context) types.Error {
	if err := sock.Close(ctx); err != nil {
		types.Fail(err)
	}

	return types.OK
}

// Send is called by the GUEST to send data to the host.
func (sock *Socket) Write(ctx context.Context, b types.Bytes, consumed types.Pointer[types.Uint32]) types.Error {
	n, err := sock.Pipe.Write(b)
	consumed.Store(types.Uint32(n))
	if err == nil {
		return types.OK
	}

	// If the read timed out, return a special error code that
	// tells the caller to back-off and try later. It is up to
	// the caller to specify the retry strategy.
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return interrupt
	}

	return types.Fail(err)
}

// Recv is called by the GUEST to receive data to the host.
func (sock *Socket) Read(ctx context.Context, b types.Bytes, size types.Pointer[types.Uint32]) types.Error {
	n, err := sock.Pipe.Read(b)
	size.Store(types.Uint32(n))
	if err == nil {
		return types.OK
	}

	// If the read timed out, return a special error code that
	// tells the caller to back-off and try later. It is up to
	// the caller to specify the retry strategy.
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return interrupt
	}

	return types.Fail(err)
}

var interrupt = types.Fail(errno(1))
