package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
	"zenhack.net/go/util/rc"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"

	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/log"
)

// module for wetware Host
var module wazergo.HostModule[*Socket] = functions{
	"__sock_recv":  wazergo.F1((*Socket).Recv),
	"__sock_send":  wazergo.F1((*Socket).Send),
	"__sock_close": wazergo.F0((*Socket).close),
}

// Socket is a system socket that uses the host's IP stack.
type Socket struct {
	Logger      log.Logger
	Host, Guest *Pipe
	Session     auth.Session

	conn *rpc.Conn
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

func (sock *Socket) dial(ctx context.Context) error {
	// NOTE:  no auth is actually performed here.  The client doesn't
	// even need to pass a valid signer; the login call always succeeds.
	server := core.Terminal_NewServer(sock.Session)
	client := capnp.NewClient(server)

	transport := hostTransport{sock}

	options := &rpc.Options{
		ErrorReporter:   ErrorReporter{Logger: sock.Logger},
		BootstrapClient: client,
	}

	sock.conn = rpc.NewConn(transport, options)
	return nil
}

func (sock *Socket) Send(ctx context.Context, buf types.Pointer[types.Bytes]) types.Error {
	msg, err := capnp.Unmarshal(buf.Load())
	if err != nil {
		return types.Fail(err)
	}

	message, err := rpccp.ReadRootMessage(msg)
	if err != nil {
		return types.Fail(err)
	}

	ref := rc.NewRef(message, msg.Release)

	if err = sock.Host.Push(ref); err != nil {
		return types.Fail(err)
	}

	return types.OK
}

func (sock *Socket) Recv(ctx context.Context, buf types.Pointer[types.Bytes]) types.Error {
	ref, err := sock.Guest.Pop()
	if err != nil {
		return types.Fail(err)
	}

	b, err := ref.Value().Message().Marshal()
	if err != nil {
		return types.Fail(err)
	}
	buf.Store(b)

	return types.OK
}
