package hostutil

import (
	p2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/pnet"
)

// MaybePrivate sets the p2p.PrivateNetwork option if the PSK is not nil.
func MaybePrivate(psk pnet.PSK) p2p.Option {
	if psk == nil {
		return p2p.ChainOptions()
	}

	return p2p.PrivateNetwork(psk)
}
