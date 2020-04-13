package ww

import (
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// EvtHeartbeat is emitted each time a peer's heartbeat is received.
type EvtHeartbeat struct {
	ID    peer.ID
	TTL   time.Duration
	Addrs []multiaddr.Multiaddr
}
