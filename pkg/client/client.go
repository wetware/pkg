// Package client exports the Wetware client API.
package client

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/ocap/anchor"
	"github.com/wetware/ww/pkg/ocap/cluster"
	"github.com/wetware/ww/pkg/ocap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

// ErrDisconnected indicates that the client's connection to
// the cluster was lost.
var ErrDisconnected = errors.New("disconnected")

type Node struct {
	vat  vat.Network
	conn *rpc.Conn

	// capabilities
	ps   pubsub.PubSub
	host cluster.Host
}

// String returns the cluster namespace
func (n Node) String() string { return n.vat.NS }

func (n Node) Loggable() map[string]interface{} {
	return n.vat.Loggable()
}

// Host returns the underlying host for the client node.
func (n Node) Host() host.Host { return n.vat.Host }

// Bootstrap blocks until the context expires, or the
// node's capabilities resolve.  It is safe to cancel
// the context passed to Dial after this method returns.
func (n Node) Bootstrap(ctx context.Context) error {
	// TODO:  update this when we replace 'ps' with a
	//        capability set.
	if err := n.ps.Client.Resolve(ctx); err != nil {
		return err
	}

	return n.host.Client.Resolve(ctx)
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n Node) Done() <-chan struct{} {
	return n.conn.Done()
}

// Close the client connection.  Note that this does not
// close the underlying host.
func (n Node) Close() error {
	n.ps.Release()

	return n.conn.Close()
}

// Join a pubsub topic.
func (n Node) Join(ctx context.Context, topic string) Topic {
	var f, release = n.ps.Join(ctx, topic)
	defer release()

	return NewTopic(f.Topic().AddRef().Client, topic)
}

func (n Node) Path() string { return "/" }

func (n Node) Ls(ctx context.Context) Iterator {
	// TODO(performance):  cache an instance of the View capability
	f, release := n.host.View(ctx, nil)
	defer release()

	it := f.View().Iter(ctx)
	runtime.SetFinalizer(it, func(it *cluster.RecordStream) {
		it.Finish()
	})

	return hostSet{
		dialer:       dialer(n.vat),
		RecordStream: it,
	}
}

func (n Node) Walk(ctx context.Context, path string) Anchor {
	p := anchor.NewPath(path)
	if p.Err() != nil {
		return newErrorHost(p.Err())
	}

	if p.IsRoot() {
		return n
	}

	p, name := p.Next()

	id, err := peer.Decode(name)
	if err != nil {
		return newErrorHost(fmt.Errorf("invalid id: %w", err))
	}

	return Host{
		dialer: dialer(n.vat),
		host:   &cluster.Host{Info: peer.AddrInfo{ID: id}},
	}.Walk(ctx, p.String())
}
