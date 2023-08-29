package server

import (
	"context"
	"fmt"
	"net"

	"log/slog"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/wetware/pkg/boot"
)

type BootConfig struct {
	NS        string
	Host      host.Host
	Discovery discovery.Discovery
}

func (conf BootConfig) Bootstrap(ctx context.Context, addr net.Addr, protos ...protocol.ID) (s network.Stream, err error) {
	var peers <-chan peer.AddrInfo
	if peers, err = conf.Discovery.FindPeers(ctx, addr.Network()); err != nil {
		return nil, fmt.Errorf("find peers: %w", err)
	}

	err = boot.ErrNoPeers
	for info := range peers {
		if s, err = conf.connect(ctx, addr, protos, info); err == nil {
			err = fmt.Errorf("%s: %w", info.ID.ShortString(), err)
			break
		}
	}

	return s, err
}

func (conf BootConfig) connect(ctx context.Context, addr net.Addr, protos []protocol.ID, info peer.AddrInfo) (network.Stream, error) {
	slog.Debug("peer discovered",
		"proto", protos,
		"peer", info)

	if err := conf.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	slog.Debug("connected to peer",
		"proto", protos,
		"peer", info)

	return conf.Host.NewStream(ctx, info.ID, protos...)
}
