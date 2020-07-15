// Package boot contains facilities for joining active clusters.
package boot

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

var _ Strategy = (StaticAddrs)(nil)

// Strategy for obtaining bootstrap peers.
type Strategy interface {
	Loggable() map[string]interface{}
	DiscoverPeers(context.Context, ...Option) (<-chan peer.AddrInfo, error)
}

// Beacon is a boot strategy that requires a service running on a local node in
// order to respond to boot requests.
type Beacon interface {
	Loggable() map[string]interface{}
	Signal(context.Context, host.Host) error
	Stop(context.Context) error
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

/*
	Misc utilities for built-in discovery strategies.
	If any of these grow into something non-trivial,
	consider exporting them in a `discutil` package.
*/

func (p *Param) isLimited() bool {
	return p.Limit > 0
}
