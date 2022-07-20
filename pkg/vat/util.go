// Package vatutil provides utilities for creating and configuring network vats.
package vat

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
)

// NewClientHost returns a libp2p.Host suitable for use in a Wetware
// client vat.  By default, the returned host uses the QUIC transport,
// and does not accept incoming network connections.
//
// Callers can override these defaults by passing libp2p.Options.
func NewClientHost(opt ...libp2p.Option) (host.Host, error) {
	return libp2p.New(withClientDefault(opt)...)
}

func withClientDefault(opt []libp2p.Option) []libp2p.Option {
	return append([]libp2p.Option{
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(quic.NewTransport),
	}, opt...)
}
