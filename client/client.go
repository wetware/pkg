package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/lthibault/log"
	ww "github.com/wetware/ww/pkg"
)

var ErrNoPeers = errors.New("no peers")

type Config struct {
	Logger   log.Logger
	NS       string
	Peers    []string // static bootstrap peers
	Discover string   // bootstrap service multiadr
}

func (cfg Config) Dial(ctx context.Context, h host.Host) (*rpc.Conn, error) {
	if cfg.Logger == nil {
		cfg.Logger = log.New()
	}
	cfg.Logger = cfg.Logger.WithField("ns", cfg.NS)

	peer, err := cfg.connect(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	s, err := h.NewStream(ctx, peer,
		ww.Subprotocol(cfg.NS),
		ww.Subprotocol(cfg.NS, "/packed"))
	if err != nil {
		return nil, fmt.Errorf("upgrade: %w", err)
	}

	return rpc.NewConn(transport(s), nil), nil
}

func (cfg Config) connect(ctx context.Context, h host.Host) (peer.ID, error) {
	d, err := cfg.newBootstrapper(h)
	if err != nil {
		return "", fmt.Errorf("bootstrap: %w", err)
	}
	defer d.Close()

	var peers <-chan peer.AddrInfo
	if peers, err = d.FindPeers(ctx, cfg.NS); err != nil {
		return "", fmt.Errorf("discover: %w", err)
	}

	for info := range peers {
		if err = h.Connect(ctx, info); err == nil {
			return info.ID, nil
		}
	}

	// no peers?
	if err == nil {
		err = ErrNoPeers
	}

	return "", err
}

func transport(s network.Stream) rpc.Transport {
	if strings.HasSuffix(string(s.Protocol()), "/packed") {
		return rpc.NewPackedStreamTransport(s)
	}

	return rpc.NewStreamTransport(s)
}
