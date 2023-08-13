package client

import (
	"context"
	"io"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
)

type VatDialer interface {
	DialVat(context.Context, local.Host) (*rpc.Conn, error)
}

// NewHost returns a local libp2p host that is suitable for
// client-only use.
func NewHost() (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

// Dial a cluster.
func Dial(ctx context.Context, h local.Host, d VatDialer) (*rpc.Conn, error) {
	conn, err := d.DialVat(ctx, h)
	if err == nil {
		runtime.SetFinalizer(conn, func(c io.Closer) error {
			return c.Close()
		})
	}

	return conn, err
}
