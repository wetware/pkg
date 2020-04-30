package server

import (
	"context"

	log "github.com/lthibault/log/pkg"
	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Host .
type Host struct {
	log          log.Logger
	host         host.Host
	routingTable interface{ Contains(peer.ID) bool }

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

// Log returns a structured logger whose fields identify the host.
func (h Host) Log() log.Logger {
	return h.log.WithFields(log.F{
		"id":    h.ID(),
		"addrs": h.Addrs(),
	})
}

// ID of the Host
func (h Host) ID() peer.ID {
	return h.host.ID()
}

// Addrs on which the host is reachable
func (h Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

// Start the Host's network connections and start its runtime processes.
func (h Host) Start() error {
	return h.app.Start(context.Background())
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	return h.app.Stop(context.Background())
}
