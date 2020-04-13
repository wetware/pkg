package randutil

import (
	"math/rand"

	peer "github.com/libp2p/go-libp2p-core/peer"
)

// FromPeer creates a random source whose seed is derived from the peer's ID.  A given
// peer's source will always be seeded with the same value, and will always produce the
// same stream, but streams will be uncorrelated across peers.
func FromPeer(id peer.ID) rand.Source {
	return rand.NewSource(quickHash(id))
}

func quickHash(id peer.ID) int64 {
	b := []byte(id)[0:8]
	return int64(b[7]) | int64(b[6])<<8 | int64(b[5])<<16 | int64(b[4])<<24 |
		int64(b[3])<<32 | int64(b[2])<<40 | int64(b[1])<<48 | int64(b[0])<<56
}
