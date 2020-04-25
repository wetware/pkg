// Package boot contains facilities for discovering bootstrap peers.
package boot

import (
	"context"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Protocol for cluster bootstrap.
type Protocol interface {
	Strategy
	Beacon
}

// Service that can be discovered.
type Service interface {
	ID() peer.ID
	Network() network.Network
}

// Strategy for obtaining bootstrap peers.
type Strategy interface {
	DiscoverPeers(context.Context) ([]peer.AddrInfo, error)
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

// DiscoverPeers converts the static addresses into AddrInfos
func (as StaticAddrs) DiscoverPeers(context.Context) (ps []peer.AddrInfo, err error) {
	return peer.AddrInfosFromP2pAddrs(as...)
}

// Start is a nop.  It immediately returns nil.
func (as StaticAddrs) Start(Service) error {
	return nil
}

// Close is a nop.  It immediately returns nil.
func (as StaticAddrs) Close() error {
	return nil
}
