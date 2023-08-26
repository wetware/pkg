// Package boot provides utilities for parsing and instantiating boot services
package boot

import (
	"errors"
	"io"

	"github.com/libp2p/go-libp2p/core/discovery"
)

var (
	// ErrUnknownBootProto is returned when the multiaddr passed
	// to Parse does not contain a recognized boot protocol.
	ErrUnknownBootProto = errors.New("unknown boot protocol")

	ErrNoPeers = errors.New("no peers")
)

type Service interface {
	discovery.Discovery
	io.Closer
}
