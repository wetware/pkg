package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/log"

	"github.com/wetware/ww/pkg/vat"
	"github.com/wetware/ww/pkg/vat/cap/cluster"
	"github.com/wetware/ww/pkg/vat/cap/pubsub"
)

// Dialer is a factory type for Node.  It uses Boot to join the
// cluster identified by Vat.NS, and returns a Node.
type Dialer struct {
	Log  log.Logger
	Vat  vat.Network
	Boot discovery.Discoverer
}

// Dial creates a client and connects it to a cluster.
func (d Dialer) Dial(ctx context.Context) (*Node, error) {
	if d.Log == nil {
		d.Log = log.New()
	}

	d.Log = d.Log.With(d.Vat)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return d.join(ctx)
}

func (d Dialer) join(ctx context.Context) (n *Node, err error) {
	var peers <-chan peer.AddrInfo
	if peers, err = d.Boot.FindPeers(ctx, d.Vat.NS); err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	for info := range peers {
		d.Log.WithField("peer_info", info).Debug("found peer")

		n, err = d.dialCaps(ctx, info)
		if err == nil {
			break
		}

		d.Log.WithError(err).Debug("failed to connect to peer")
	}

	// no peers discovered?
	if n == nil && err == nil {
		err = errors.New("bootstrap failed: no peers found")
	}

	return
}

func (d Dialer) dialCaps(ctx context.Context, info peer.AddrInfo) (*Node, error) {
	psConn, err := d.Vat.Connect(ctx, info, pubsub.Capability)
	if err != nil {
		return nil, err
	}

	hostConn, err := d.Vat.Connect(ctx, info, cluster.HostCapability)
	if err != nil {
		return nil, err
	}

	return &Node{
		vat:  d.Vat,
		conn: hostConn, // TODO:  do we still need an rpc.Conn?  Should we prefer one conn over the other?
		ps:   pubsub.PubSub{Client: psConn.Bootstrap(context.Background())},
		host: cluster.Host{Client: hostConn.Bootstrap(context.Background())},
	}, nil
}
