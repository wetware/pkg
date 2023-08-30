package client

import (
	"context"
	"fmt"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"

	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/util/proto"
)

type Dialer struct {
	Host local.Host
	// Account auth.Signer
}

func (d Dialer) DialDiscover(ctx context.Context, ds discovery.Discoverer, ns string) (host.Host, error) {
	peers, err := ds.FindPeers(ctx, ns)
	if err != nil {
		return host.Host{}, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if err = d.Host.Connect(ctx, info); err != nil {
			continue
		}

		return d.Dial(ctx, info, proto.Namespace(ns)...)
	}

	return host.Host{}, err
}

func (d Dialer) Dial(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (host.Host, error) {
	conn, err := d.DialRPC(ctx, addr, protos...)
	if err != nil {
		return host.Host{}, fmt.Errorf("dial: %w", err)
	}

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return host.Host{}, fmt.Errorf("bootstrap: %w", err)
	}

	return host.Host(client), nil
}

func (d Dialer) DialRPC(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (*rpc.Conn, error) {
	s, err := d.DialP2P(ctx, addr, protos...)
	if err != nil {
		return nil, err
	}

	conn := rpc.NewConn(transport(s), nil)

	return conn, nil
}

func (d Dialer) DialP2P(ctx context.Context, addr peer.AddrInfo, protos ...protocol.ID) (network.Stream, error) {
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
