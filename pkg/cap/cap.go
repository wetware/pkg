// Package cap exports abstractions for working with object capabilities.
package cap

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/network"
)

type StreamBinder interface {
	BindStream(network.Stream) (*capnp.Client, error)
}

type NewBinder func(network.Stream) (*capnp.Client, error)

func (bind NewBinder) BindStream(s network.Stream) (*capnp.Client, error) {
	return bind(s)
}

type Dialer interface {
	Dial(context.Context, StreamBinder) (*capnp.Client, error)
}
