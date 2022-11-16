package ww

import (
	"context"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/lthibault/log"
	ma "github.com/multiformats/go-multiaddr"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
)

// Dial into the cluster designated by ns, using the supplied
// bootstrap peer and dialer options.   For more control over
// cluster dialing and client-node construction, see 'Dialer'
// in pkg/client.
func Dial(ctx context.Context, ns string, peer peer.AddrInfo, opt ...DialOpt) (*client.Node, error) {
	return DialDiscover(ctx, ns, boot.StaticAddrs{peer})
}

// DialAddr dials the cluster designated by ns, using a multiaddr.
// The supplied multiaddr is converted into a peer.AddrInfo before
// being passed to Dial.
func DialAddr(ctx context.Context, ns string, ma ma.Multiaddr, opt ...DialOpt) (*client.Node, error) {
	info, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, err
	}

	return Dial(ctx, ns, *info)
}

// DialString behaves like DialAddr, but accepts a string-encoded
// multiaddr.  It is provided as a convenience.
func DialString(ctx context.Context, ns, addr string, opt ...DialOpt) (*client.Node, error) {
	maddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}

	return DialAddr(ctx, ns, maddr)
}

// DialDiscover connects to the cluster designated by ns, using the
// supplied discovery service.
func DialDiscover(ctx context.Context, ns string, d discovery.Discoverer, opt ...DialOpt) (*client.Node, error) {
	vat, err := casm.New(ns, casm.Client())
	if err != nil {
		return nil, err
	}

	dialer := client.Dialer{Vat: vat, Boot: d}
	for _, option := range opt {
		option(&dialer)
	}

	return dialer.Dial(ctx)
}

// DialOpt is an option type that can be passed to dialer
// functions in this package.   It is an abstraction over
// the Dialer type in pkg/client, intended to facilitate
// common boilerplate.  For more flexibility, use package
// pkg/client directly.
type DialOpt func(*client.Dialer)

// WithLogger assigns the supplied logger to the cluster
// dialer and the client.Node it constructs. If l == nil,
// a default logger is used.
func WithLogger(l log.Logger) DialOpt {
	if l == nil {
		l = log.New()
	}

	return func(d *client.Dialer) {
		d.Log = l
	}
}
