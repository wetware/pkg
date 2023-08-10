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
	"github.com/wetware/pkg/util/proto"
)

var ErrNoPeers = errors.New("no peers")

type Dialer struct {
	Logger   log.Logger
	NS       string
	Peers    []string // static bootstrap peers
	Discover string   // bootstrap service multiadr
}

func (d Dialer) Dial(ctx context.Context, h host.Host) (*rpc.Conn, error) {
	if d.Logger == nil {
		d.Logger = log.New()
	}
	d.Logger = d.Logger.WithField("ns", d.NS)

	peer, err := d.connect(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// Get a set of Wetware subprotocols that we can try to dial.   These
	// will negotiate things like Cap'n Proto schema version, Cap'n Proto
	// bit-packing and LZ4 compression.
	protos := proto.Namespace(d.NS)

	s, err := h.NewStream(ctx, peer, protos...)
	if err != nil {
		return nil, fmt.Errorf("upgrade: %w", err)
	}

	return rpc.NewConn(transport(s), nil), nil
}

func (d Dialer) connect(ctx context.Context, h host.Host) (peer.ID, error) {
	boot, err := d.newBootstrapper(h)
	if err != nil {
		return "", fmt.Errorf("bootstrap: %w", err)
	}
	defer boot.Close()

	var peers <-chan peer.AddrInfo
	if peers, err = boot.FindPeers(ctx, d.NS); err != nil {
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
