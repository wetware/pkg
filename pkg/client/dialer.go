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
	ww "github.com/wetware/ww/pkg"
)

var ErrNoPeers = errors.New("no peers")

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

		if n, err = d.connect(ctx, info); err == nil {
			break
		}

		d.Log.WithError(err).Debug("failed to connect to peer")
	}

	// no peers discovered?
	if n == nil && err == nil {
		err = ErrNoPeers
	}

	return
}

func (d Dialer) connect(ctx context.Context, info peer.AddrInfo) (*Node, error) {
	conn, err := d.Vat.Connect(ctx, info, ww.HostCapability)
	if err != nil {
		return nil, err
	}

	return &Node{
		Vat:  d.Vat,
		conn: conn,
	}, nil
}

func addrEntry(info peer.AddrInfo) log.F {
	return log.F{
		"peer":  info.ID,
		"addrs": info.Addrs,
	}
}
