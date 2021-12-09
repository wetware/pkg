package client

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"go.uber.org/fx"

	"github.com/wetware/ww/pkg/cap"
)

type BasicCapDialer struct {
	fx.In

	NS   string `optional:"true" name:"ns"`
	D    discovery.Discoverer
	Opts []discovery.Option

	Host   host.Host
	Protos []protocol.ID
}

func (cd BasicCapDialer) Dial(ctx context.Context, bind cap.StreamBinder) (*capnp.Client, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	peers, err := cd.D.FindPeers(ctx, cd.NS, cd.Opts...)
	if err != nil {
		return nil, DiscoveryError{
			NS:    cd.NS,
			Cause: err,
		}
	}

	var (
		c    *capnp.Client
		info peer.AddrInfo
	)

	for info = range peers {
		if c, err = cd.dial(ctx, info, bind); err == nil {
			return c, nil
		}
	}

	// exited loop due to context expiration?
	if err == nil {
		return nil, DiscoveryError{
			NS:    cd.NS,
			Cause: ctx.Err(),
		}
	}

	return nil, DialError{
		NS:    cd.NS,
		Peer:  &info,
		Cause: err,
	}
}

func (cd BasicCapDialer) dial(ctx context.Context, info peer.AddrInfo, b cap.StreamBinder) (*capnp.Client, error) {
	if err := cd.Host.Connect(ctx, info); err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	s, err := cd.Host.NewStream(ctx, info.ID, cd.Protos...)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	return b.BindStream(s)
}

type DiscoveryError struct {
	NS    string
	Cause error
}

func (err DiscoveryError) Error() string {
	return fmt.Sprintf("(%s) find peers: %s",
		err.NS,
		err.Cause)
}

type DialError struct {
	NS    string
	Peer  *peer.AddrInfo
	Cause error
}

func (err DialError) Error() string {
	return fmt.Sprintf("(%s) dial %s: %s",
		err.NS,
		err.Peer.ID.ShortString(),
		err.Cause)
}
