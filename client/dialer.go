package client

import (
	"context"
	"fmt"
	"net"
	"strings"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/util/proto"
)

type Addr struct {
	net.Addr
	Protos []protocol.ID
}

// Bootstrapper can resolve
type Bootstrapper interface {
	Bootstrap(context.Context, *Addr) (network.Stream, error)
}

type Dialer[T ~capnp.ClientKind] struct {
	Bootstrapper Bootstrapper
	Auth         auth.Policy[T]
	Opts         *rpc.Options
}

func (d Dialer[T]) Dial(ctx context.Context, addr *Addr) (auth.Session[T], error) {
	conn, err := d.DialRPC(ctx, addr)
	if err != nil {
		return auth.DenyAll[T](), fmt.Errorf("dial: %w", err)
	}

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.DenyAll[T](), fmt.Errorf("bootstrap: %w", err)
	}

	return d.Auth.Login(ctx, T(client)), nil
}

func (d Dialer[T]) DialRPC(ctx context.Context, addr net.Addr) (*rpc.Conn, error) {
	peer := &Addr{
		Addr:   addr,
		Protos: proto.Namespace(addr.Network()),
		// Get a set of Wetware subprotocols that we can try to dial.   These
		// will negotiate things like Cap'n Proto schema version, Cap'n Proto
		// bit-packing and LZ4 compression.
	}

	s, err := d.Bootstrapper.Bootstrap(ctx, peer)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(transport(s), nil)
	return conn, nil
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
