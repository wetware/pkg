// Package boot contains facilities for joining active clusters.
package boot

import (
	"context"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

var _ Strategy = (StaticAddrs)(nil)

// // Protocol for cluster bootstrap.
// type Protocol interface {
// 	Strategy
// 	Beacon
// }

// Service that can be discovered.
type Service interface {
	ID() peer.ID
	Network() network.Network
}

// Strategy for obtaining bootstrap peers.
type Strategy interface {
	Loggable() map[string]interface{}
	DiscoverPeers(context.Context, ...Option) (<-chan peer.AddrInfo, error)
}

// Beacon responds to queries from a corresponding Discover implementation.
// Implementations MUST ensure Start can be called again after a successful call to
// Close, and SHOULD make efforts to ensure Close returns in a timely manner.
type Beacon interface {
	// Start advertising the service in the background.  Does not block.
	// Subsequent calls to Start MUST be preceeded by a call to Close.
	Start(Service) error

	// Close stops the active service advertisement.  Once called, Start can be called
	// again.
	Close() error
}

// StaticAddrs for cluster discovery
type StaticAddrs []multiaddr.Multiaddr

// Loggable representation
func (as StaticAddrs) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"boot_strategy": "static_addrs",
		"boot_addrs":    as,
	}
}

// DiscoverPeers converts the static addresses into AddrInfos
func (as StaticAddrs) DiscoverPeers(_ context.Context, opt ...Option) (<-chan peer.AddrInfo, error) {
	var p Param
	if err := p.Apply(opt); err != nil {
		return nil, err
	}

	if p.isLimited() && len(as) > p.Limit {
		as = as[:p.Limit]
	}

	ps, err := peer.AddrInfosFromP2pAddrs(as...)
	if err != nil {
		return nil, err
	}

	ch := make(chan peer.AddrInfo, len(ps))
	for _, p := range ps {
		ch <- p
	}
	close(ch)

	return ch, err
}

// Start is a nop.  It immediately returns nil.
func (as StaticAddrs) Start(Service) error {
	return nil
}

// Close is a nop.  It immediately returns nil.
func (as StaticAddrs) Close() error {
	return nil
}

/*
	Misc utilities for built-in discovery strategies.
	If any of these grow into something non-trivial,
	consider exporting them in a `discutil` package.
*/

func (p *Param) isLimited() bool {
	return p.Limit > 0
}