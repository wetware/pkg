package server

import (
	"context"
	"fmt"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/client"
)

type BootConfig struct {
	NS        string
	Host      host.Host
	Discovery discovery.Discovery
	Peers     []string
}

func (conf BootConfig) Bootstrap(ctx context.Context, addr *client.Addr) (s network.Stream, err error) {
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
		if s, err = conf.connect(ctx, addr, info); err == nil {
			err = fmt.Errorf("%s: %w", info.ID.ShortString(), err)
			break
		}
	}

	return s, err
}

func (conf BootConfig) discovery() (_ discovery.Discovery, err error) {
	// use discovery service?
	if len(conf.Peers) == 0 {
		return conf.Discovery, nil // slow
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

func (conf BootConfig) connect(ctx context.Context, addr *client.Addr, info peer.AddrInfo) (network.Stream, error) {
	if err := conf.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return conf.Host.NewStream(ctx, info.ID, addr.Protos...)
}
