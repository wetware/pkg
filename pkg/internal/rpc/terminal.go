package rpc

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"

	capnp "zombiezen.com/go/capnproto2"
)

// Client tags a capnp.Client with the remote endpoint's peer.ID.
type Client struct {
	Peer peer.ID
	*capnp.Client
}

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

// NewTerminal .
func NewTerminal(h host.Host) Terminal {
	return Terminal{
		Host: h,
	}
}

// Dial a method on a remote host
func (t Terminal) Dial(ctx context.Context, d Dialer, pids ...protocol.ID) Client {
	return d.Dial(ctx, streamCachingHost(t), pids)
}

// HangUp the client, freeing its resources for reuse.
func (t Terminal) HangUp(c Client) {
	/*
		TODO(performance):  caching
	*/

	c.Release()
}

// Session binds a client to a Terminal, allowing it to be returned to a free list when
// no longer needed.
type Session struct {
	Client
}

type streamCachingHost Terminal

// NewStream overrides Host.NewStream, using cached results
func (h streamCachingHost) NewStream(ctx context.Context, id peer.ID, pids ...protocol.ID) (network.Stream, error) {
	/*
		TODO(performance) caching goes here
	*/
	return h.Host.NewStream(ctx, id, pids...)
}
