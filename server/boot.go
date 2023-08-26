package server

import (
	"context"
	"fmt"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/client"
)

type Net interface {
	Network() string
	discovery.Discovery
}

type BootConfig struct {
	Net
	Host  host.Host
	Peers []string
	RPC   *rpc.Options
}

func (conf BootConfig) Bootstrap(ctx context.Context, addr *client.Addr) (conn *rpc.Conn, err error) {
	var d discovery.Discovery
	if d, err = conf.discovery(); err != nil {
		return nil, fmt.Errorf("discovery: %w", err)
	}

	var peers <-chan peer.AddrInfo
	if peers, err = d.FindPeers(ctx, addr.Network()); err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if conn, err = conf.connect(ctx, addr, info); err == nil {
			err = fmt.Errorf("%s: %w", info.ID.ShortString(), err)
			break
		}
	}

	return conn, err
}

func (conf BootConfig) discovery() (_ discovery.Discovery, err error) {
	// use discovery service?
	if len(conf.Peers) == 0 {
		return conf.Net, nil // slow
	}

	// fast path; direct dial a peer
	maddrs := make([]ma.Multiaddr, len(conf.Peers))
	for i, s := range conf.Peers {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}

func (conf BootConfig) connect(ctx context.Context, addr *client.Addr, info peer.AddrInfo) (*rpc.Conn, error) {
	if err := conf.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	stream, err := conf.Host.NewStream(ctx, info.ID, addr.Protos...)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	return rpc.NewConn(transport(stream), conf.RPC), nil
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
