package ww

import (
	"context"

	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/event"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	log "github.com/lthibault/log/pkg"
	service "github.com/lthibault/service/pkg"
)

// StreamAPI .
type StreamAPI interface {
	// SetStreamHandler sets the protocol handler on the Host's Mux.
	// (Thread-safe)
	SetStreamHandler(protocol.ID, network.StreamHandler)

	// SetStreamHandlerMatch sets the protocol handler on the Host's Mux
	// using a matching function for protocol selection.
	SetStreamHandlerMatch(protocol.ID, func(string) bool, network.StreamHandler)

	// RemoveStreamHandler removes a handler on the mux that was set by
	// SetStreamHandler
	RemoveStreamHandler(protocol.ID)

	// NewStream opens a new stream to given peer p, and writes a p2p/protocol
	// header with given ProtocolID. If there is no connection to p, attempts
	// to create one. If ProtocolID is "", writes no header.
	// (Thread-safe)
	NewStream(context.Context, peer.ID, ...protocol.ID) (network.Stream, error)
}

// Host .
type Host struct {
	log  log.Logger
	root service.Service

	iface.CoreAPI
	host host.Host
}

// New Host
// TODO:  Options, e.g. for the repo path or whatever
func New(opt ...Option) (*Host, error) {
	var h = new(Host)
	/*
		Host is instantiated via dependency injection.  It's easy as:

		1. Create a Runtime for the Host.
		2. Bind the runtime to the host, obtaining a service.

		The service can be freely configured using github.com/lthibault/service
	*/

	r := new(Runtime)
	if err := r.setOptions(opt); err != nil {
		return nil, err
	}

	if err := r.Verify(); err != nil {
		return nil, err
	}

	r.Bind(h)

	return h, nil
}

// Start the Host's network connections and start its runtime processes.
func (h Host) Start() error {
	return h.root.Start()
}

// Close the Host's network connections and stop its runtime processes.
func (h Host) Close() error {
	return h.root.Stop()
}

// Stream API
func (h Host) Stream() StreamAPI {
	return h.host
}

// EventBus API
func (h Host) EventBus() event.Bus {
	return h.host.EventBus()
}
