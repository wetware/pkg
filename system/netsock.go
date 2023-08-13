package system

import (
	"context"
	"net"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/pkg/util/log"
)

// NetSock is a system socket that uses the host's IP stack.
type NetSock struct {
	Addr            net.Addr
	Logger          log.Logger
	BootstrapClient capnp.Client

	conn *rpc.Conn
}

func (sock *NetSock) Close(context.Context) error {
	sock.BootstrapClient.Release()

	return sock.conn.Close()
}

func (sock *NetSock) dial(ctx context.Context) error {
	raw, err := dial(ctx, sock.Addr)
	if err != nil {
		return err
	}

	sock.conn = rpc.NewConn(rpc.NewStreamTransport(raw), &rpc.Options{
		ErrorReporter:   ErrorReporter{Logger: sock.Logger},
		BootstrapClient: sock.BootstrapClient,
	})

	return nil
}

func dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, addr.Network(), addr.String())
}
