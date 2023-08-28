package client

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3/rpc"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/wetware/pkg/boot"
)

type Net interface {
	Network() string
}

type BootConfig struct {
	NS        string
	Discovery discovery.Discovery
	Host      local.Host
	Peers     []string
	RPC       *rpc.Options
}

func (conf BootConfig) Bootstrap(ctx context.Context, addr *Addr) (s network.Stream, err error) {
	var d discovery.Discoverer
	if d, err = conf.discovery(); err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	var peers <-chan peer.AddrInfo
	if peers, err = d.FindPeers(ctx, addr.Network()); err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if s, err = conf.dial(ctx, addr, info); err == nil {
			break
		}
	}

	return s, err
}

func (conf BootConfig) discovery() (_ discovery.Discoverer, err error) {
	if len(conf.Peers) == 0 {
		return conf.Discovery, nil
	}

	maddrs := make([]ma.Multiaddr, len(conf.Peers))
	for i, s := range conf.Peers {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}

func (conf BootConfig) dial(ctx context.Context, addr *Addr, info peer.AddrInfo) (network.Stream, error) {
	if err := conf.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return conf.Host.NewStream(ctx, info.ID, addr.Protos...)
}
