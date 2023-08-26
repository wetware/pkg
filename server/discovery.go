package server

import (
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/routing"
	disc_util "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// bootstrap discovery is the lowest-level peer discovery.  It is an
// abstraction over an out-of-band protocol that delivers a small number
// of peers.  The simplest example of this is boot.StaticAddrs.
func (conf Config) bootstrap() discovery.Discovery {
	return trimPrefix{withLogging{conf.Discovery}}
}

// ambient peer discovery represents the ability of a peer to enumerate
// peers via gossip.  This is generally much more efficient than bootstrap
// discovery.  Most implementations rely on the DHT.
func (conf Config) ambient(r routing.ContentRouting) discovery.Discovery {
	return disc_util.NewRoutingDiscovery(r)
}
