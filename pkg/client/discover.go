package client

import (
	"context"
	"net"

	"github.com/pkg/errors"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/whyrusleeping/mdns"
)

func init() {
	// logs produce false-positive errors.
	mdns.DisableLogging = true
}

// Discover is an abstract strategy for ambient peer discovery.
type Discover interface {
	DiscoverPeers(context.Context) ([]peer.AddrInfo, error)
}

// StaticAddrs for cluster discovery
type StaticAddrs []multiaddr.Multiaddr

// DiscoverPeers converts the static addresses into AddrInfos
func (as StaticAddrs) DiscoverPeers(context.Context) (ps []peer.AddrInfo, err error) {
	return peer.AddrInfosFromP2pAddrs(as...)
}

// MDNSDiscovery discovers ambient peers through multicast DNS (RFC 6762)
type MDNSDiscovery struct {
	Interface *net.Interface
}

// DiscoverPeers queries MDNS.
func (d MDNSDiscovery) DiscoverPeers(ctx context.Context) ([]peer.AddrInfo, error) {
	entries := make(chan *mdns.ServiceEntry, 1)

	if err := mdns.Query(&mdns.QueryParam{
		Service:             discovery.ServiceTag,
		Entries:             entries,
		Interface:           d.Interface,
		WantUnicastResponse: true,
	}); err != nil {
		return nil, errors.Wrap(err, "mdns query")
	}

	select {
	case entry := <-entries:
		return d.handleEntry(entry)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (d MDNSDiscovery) handleEntry(e *mdns.ServiceEntry) ([]peer.AddrInfo, error) {
	mpeer, err := peer.IDB58Decode(e.Info)
	if err != nil {
		return nil, errors.Wrap(err, "decode b58")
	}

	maddr, err := manet.FromNetAddr(&net.TCPAddr{IP: e.AddrV4, Port: e.Port})
	if err != nil {
		return nil, errors.Wrap(err, "parse multiaddr")
	}

	return []peer.AddrInfo{
		{ID: mpeer, Addrs: []multiaddr.Multiaddr{maddr}},
	}, nil
}
