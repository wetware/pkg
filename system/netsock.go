package system

import (
	"context"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/log"
)

// NetSock is a system socket that uses the host's IP stack.
type NetSock struct {
	Addr    net.Addr
	Logger  log.Logger
	Session auth.Session

	conn *rpc.Conn
}

func (sock *NetSock) Close(context.Context) error {
	sock.Session.Release()

	return sock.conn.Close()
}

func (sock *NetSock) dial(ctx context.Context) error {
	raw, err := dial(ctx, sock.Addr)
	if err != nil {
		return err
	}

	// NOTE:  no auth is actually performed here.  The client doesn't
	// even need to pass a valid signer; the login call always succeeds.
	server := core.Terminal_NewServer(sock.Session)
	client := capnp.NewClient(server)

	sock.conn = rpc.NewConn(rpc.NewStreamTransport(raw), &rpc.Options{
		ErrorReporter:   ErrorReporter{Logger: sock.Logger},
		BootstrapClient: client,
	})

	return nil
}

func dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, addr.Network(), addr.String())
}
