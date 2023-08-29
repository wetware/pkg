package client

import (
	"context"
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p/core/discovery"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/wetware/pkg/boot"
)

type BootConfig struct {
	NS         string
	Host       local.Host
	Discoverer discovery.Discoverer
}

func (conf BootConfig) Bootstrap(ctx context.Context, addr net.Addr, protos ...protocol.ID) (s network.Stream, err error) {
	var peers <-chan peer.AddrInfo
	if peers, err = conf.Discoverer.FindPeers(ctx, addr.Network()); err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if s, err = conf.dial(ctx, addr, protos, info); err == nil {
			break
		}
	}

	return s, err
}

func (conf BootConfig) dial(ctx context.Context, addr net.Addr, protos []protocol.ID, info peer.AddrInfo) (network.Stream, error) {
	if err := conf.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	return conf.Host.NewStream(ctx, info.ID, protos...)
}
