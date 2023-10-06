package system

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	local "github.com/libp2p/go-libp2p/core/host"

	"capnproto.org/go/capnp/v3/rpc"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/util/log"
)

var SocketModule wazergo.HostModule[*Socket] = functions{
	"_sysread":  wazergo.F2((*Socket).Read),
	"_syswrite": wazergo.F2((*Socket).Write),
	"_sysclose": wazergo.F0((*Socket).close),
}

type Option = wazergo.Option[*Socket]

func WithLogger(log log.Logger) Option {
	return wazergo.OptionFunc(func(sock *Socket) {
		sock.Logger = log
	})
}

type Bindable interface {
	Bind(*Socket) *rpc.Conn
}

func Bind(b Bindable) Option {
	return wazergo.OptionFunc(func(sock *Socket) {
		sock.conn = b.Bind(sock)
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
	host, guest := net.Pipe()
	sock := &Socket{
		Host:  host,
		Guest: guest,
	}

	wazergo.Configure(sock, opts...)
	return
}

// Socket is a system socket that uses the host's IP stack.
type Socket struct {
	Logger      log.Logger
	Net         local.Host
	Name        string
	View        cluster.View
	Host, Guest net.Conn
	conn        *rpc.Conn
}

func (sock *Socket) Login(ctx context.Context, call core.Terminal_login) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	sess, err := res.NewSession()
	if err != nil {
		return err
	}

	_ = sess.Local().SetHost(sock.Name)
	_ = sess.Local().SetPeer(string(sock.Net.ID()))

	return sess.SetView(sock.View)
}

func (sock *Socket) Shutdown() {
	if sock.conn != nil {
		if err := sock.conn.Close(); err != nil {
			// Our in-process net.Conn MUST successfully close.   This allows us to
			// guarantee that no access to the object exists after Shutdown returns.
			panic(err)
		}
	}
}

func (sock *Socket) Close(context.Context) (err error) {
	if sock != nil && sock.conn != nil {
		err = sock.conn.Close()
	}

	return
}

func (sock *Socket) close(ctx context.Context) types.Error {
	if err := sock.Close(ctx); err != nil {
		types.Fail(err)
	}

	return types.OK
}

// Send is called by the GUEST to send data to the host.
func (sock *Socket) Write(ctx context.Context, b types.Bytes, consumed types.Pointer[types.Uint32]) types.Error {
	spew.Dump(sock)

	deadline := time.Now().Add(time.Millisecond)
	if err := sock.Guest.SetWriteDeadline(deadline); err != nil {
		return types.Fail(err)
	}

	slog.Warn("system.Socket.Write()") // XXX: DEBUG

	n, err := sock.Guest.Write(b)
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
	spew.Dump(sock)

	deadline := time.Now().Add(time.Millisecond)
	if err := sock.Guest.SetReadDeadline(deadline); err != nil {
		return types.Fail(err)
	}

	n, err := sock.Guest.Read(b)
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

type errno int32

func (errno) Error() string {
	return os.ErrDeadlineExceeded.Error()
}

func (e errno) Errno() int32 {
	return int32(e)
}
