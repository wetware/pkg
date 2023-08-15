package system

import (
	"context"
	"net"

	"capnproto.org/go/capnp/v3/rpc"
	"golang.org/x/exp/slog"
)

// NetSock is a system socket that uses the host's IP stack.
type NetSock struct {
	Addr net.Addr
	Opt  rpc.Options

	conn *rpc.Conn
}

func (sock *NetSock) Close(context.Context) error {
	slog.Warn("sock.Close(ctx): ...")
	return sock.conn.Close()
}

func (sock *NetSock) dial(ctx context.Context) error {
	conn, err := dial(ctx, sock.Addr)
	if err == nil {
		sock.conn = sock.upgrade(conn)
	}

	return err
}

func (sock *NetSock) upgrade(conn net.Conn) *rpc.Conn {
	return rpc.NewConn(rpc.NewStreamTransport(conn), &sock.Opt)
}

func dial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, addr.Network(), addr.String())
}
