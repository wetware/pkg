package server

import (
	"context"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Host .
type Host struct {
	logger

	core interface {
		ID() peer.ID
		Addrs() []multiaddr.Multiaddr
	}

	routingTable interface {
		Contains(peer.ID) bool
	}

	app interface {
		Start(context.Context) error
		Stop(context.Context) error
	}
}

// New Host
func New(opt ...Option) Host {
	var h Host
	h.app = fx.New(module(&h, opt))
	return h
}

// ID of the Host
func (h Host) ID() peer.ID {
	return h.core.ID()
}

// Addrs on which the host is reachable
func (h Host) Addrs() []multiaddr.Multiaddr {
	return h.core.Addrs()
}

// Start the Host's network connections and start its runtime processes.
func (h Host) Start() error {
	return h.app.Start(context.Background())
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	return h.app.Stop(context.Background())
}
