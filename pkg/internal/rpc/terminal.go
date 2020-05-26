package rpc

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
)

// Terminal provides a low-level API for interacting with remote hosts over RPC.
// It abstracts over details of connecting to remote hosts and obtaining references to
// core capabilities.
//
// Use Terminal when using a single Host to provide cluster-level capabilities.
//
// TODO(performance):  Resource cacheing is in-scope and will be added in the future.
type Terminal struct {
	host.Host
}

// Call a method on a remote host
func (t Terminal) Call(ctx context.Context, d Dialer, c Caller) {
	client := d.Dial(ctx, streamCachingHost(t), c.Protocol())
	// defer t.Hangup(c)

	c.HandleRPC(ctx, client)
}

type streamCachingHost Terminal

// NewStream overrides Host.NewStream, using cached results
func (h streamCachingHost) NewStream(ctx context.Context, id peer.ID, pids ...protocol.ID) (network.Stream, error) {
	/*
		TODO(performance) caching goes here
	*/
	return h.Host.NewStream(ctx, id, pids...)
}
