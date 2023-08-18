package multicast

import (
	"context"
	"fmt"
	"net"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
)

const MaxDatagramSize = 8192

type Socket struct {
	net.PacketConn
}

func Bind(addr *net.UDPAddr) (*Socket, error) {
	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("listen multicast %s: %w",
			addr.Network(),
			err)
	}
	if err = conn.SetReadBuffer(MaxDatagramSize); err != nil {
		return nil, fmt.Errorf("set read buffer: %w", err)
	}

	return &Socket{
		PacketConn: conn,
	}, nil
}

func (sock Socket) Send(ctx context.Context, message []byte) error {
	t, _ := ctx.Deadline()
	if err := sock.SetWriteDeadline(t); err != nil {
		return err
	}

	_, err := sock.WriteTo(message, sock.LocalAddr())
	return err
}

func (sock Socket) Recv(ctx context.Context) ([]byte, error) {
	t, _ := ctx.Deadline()
	if err := sock.SetReadDeadline(t); err != nil {
		return nil, err
	}

	buf := bufferpool.Default.Get(MaxDatagramSize)

	n, _, err := sock.ReadFrom(buf)
	if err != nil {
		defer bufferpool.Default.Put(buf)
		return nil, err
	}

	return buf[:n], nil
}
