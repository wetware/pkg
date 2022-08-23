package client

import (
	"context"
	"errors"
	"fmt"

	"github.com/lthibault/log"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww/pkg/cluster"
)

// Dialer is a factory type for Node.  It uses Boot to join the
// cluster identified by Vat.NS, and returns a Node.
type Dialer struct {
	fx.In

	Log  log.Logger
	Vat  casm.Vat
	Boot discovery.Discoverer
}

// Dial creates a client and connects it to a cluster.
func (d Dialer) Dial(ctx context.Context) (*Node, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if d.Log == nil {
		d.Log = log.New()
	}

	return d.join(ctx)
}

func (d Dialer) join(ctx context.Context) (n *Node, err error) {
	var peers <-chan peer.AddrInfo
	if peers, err = d.Boot.FindPeers(ctx, d.Vat.NS); err != nil {
		return nil, fmt.Errorf("discover: %w", err)
	}

	for info := range peers {
		d.Log.With(addrEntry(info)).Debug("found peer")

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
	// psConn, err := d.Vat.Connect(ctx, info, pubsub.Capability)
	// if err != nil {
	// 	return nil, err
	// }

	hostConn, err := d.Vat.Connect(ctx, info, cluster.HostCapability)
	if err != nil {
		return nil, err
	}

	return &Node{
		vat:  d.Vat,
		conn: hostConn, // TODO:  do we still need an rpc.Conn?  Should we prefer one conn over the other?
		// ps:   pubsub.PubSub(psConn.Bootstrap(context.Background())),
		host: cluster.Host{Client: hostConn.Bootstrap(context.Background())},
	}, nil
}

func addrEntry(info peer.AddrInfo) log.F {
	return log.F{
		"peer":  info.ID,
		"addrs": info.Addrs,
	}
}
