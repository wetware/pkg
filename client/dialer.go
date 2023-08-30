package client

import (
	"context"
	"fmt"
	"strings"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/util/proto"
)

type Dialer[T ~capnp.ClientKind] struct {
	Host    local.Host
	Account auth.Signer
}

func (d Dialer[T]) DialDiscover(ctx context.Context, ds discovery.Discoverer, ns string) auth.Session[T] {
	peers, err := ds.FindPeers(ctx, ns)
	if err != nil {
		return auth.Failf[T]("find peers: %w", err)
	}

	var sess = auth.Fail[T](boot.ErrNoPeers)
	for info := range peers {
		if err := d.Host.Connect(ctx, info); err != nil {
			sess = auth.Fail[T](err)
			continue
		}

		return d.Dial(ctx, info, proto.Namespace(ns)...)
	}

	return sess
}

func (d Dialer[T]) Dial(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) auth.Session[T] {
	conn, err := d.DialRPC(ctx, addr, protos...)
	if err != nil {
		return auth.Failf[T]("dial: %w", err)
	}

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Failf[T]("bootstrap: %w", err)
	}

	return auth.Session[T]{
		Client: T(client),
	}
}

func (d Dialer[T]) DialRPC(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (*rpc.Conn, error) {
	s, err := d.DialP2P(ctx, addr, protos...)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(transport(s), &rpc.Options{
		BootstrapClient: d.Account.Client(),
	})

	return conn, nil
}

func (d Dialer[T]) DialP2P(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (network.Stream, error) {
	if len(addr.Addrs) > 0 {
		if err := d.Host.Connect(ctx, addr); err != nil {
			return nil, fmt.Errorf("dial %s: %w", addr.ID, err)
		}
	}

	return d.Host.NewStream(ctx, addr.ID, protos...)
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
