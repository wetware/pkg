package server

import (
	"context"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Host .
type Host struct {
	logger
	rt   routingTable
	host host.Host

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
	return h.host.ID()
}

// Addrs on which the host is reachable
func (h Host) Addrs() []multiaddr.Multiaddr {
	return h.host.Addrs()
}

// Peers in the cluster
func (h Host) Peers() peer.IDSlice {
	return append(h.rt.Peers(), h.host.ID())
}

// EventBus provides asynchronous notifications of changes in the host's internal state,
// or the state of the environment.
func (h Host) EventBus() event.Bus {
	return h.host.EventBus()
}

// Start the Host's network connections and start its runtime processes.
func (h Host) Start() error {
	return h.app.Start(context.Background())
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	return h.app.Stop(context.Background())
}
