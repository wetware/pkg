package system

import (
	"context"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/stealthrocket/wazergo/types"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/log"
)

// Socket is a system socket that uses the host's IP stack.
type Socket struct {
	Logger  log.Logger
	Conn    net.Conn
	Session auth.Session

	conn *rpc.Conn
}

func (sock *Socket) Close(context.Context) error {
	sock.Session.Release()

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

	sock.conn = rpc.NewConn(rpc.NewStreamTransport(sock.Conn), &rpc.Options{
		ErrorReporter:   ErrorReporter{Logger: sock.Logger},
		BootstrapClient: client,
	})

	return nil
}
