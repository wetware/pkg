// Package client exports the Wetware client API.
package client

import (
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww/pkg/cluster"
)

// ErrDisconnected indicates that the client's connection to
// the cluster was lost.
var ErrDisconnected = errors.New("disconnected")

type Node struct {
	casm.Vat

	once sync.Once
	conn *rpc.Conn
	host cluster.Host
}

// Bootstrap blocks until the context expires, or the
// node's capabilities resolve.  It is safe to cancel
// the context passed to Dial after this method returns.
func (n *Node) Bootstrap(ctx context.Context) error {
	n.once.Do(func() {
		n.host = cluster.Host(n.conn.Bootstrap(ctx))
	})

	return capnp.Client(n.host).Resolve(ctx)
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n *Node) Done() <-chan struct{} {
	return n.conn.Done()
}

// Close the client connection.  Note that this does not
// close the underlying host.
func (n *Node) Close() error {
	n.host.Release()

	return n.conn.Close()
}

// // Join a pubsub topic.
// func (n *Node) Join(ctx context.Context, topic string) Topic {
// 	var f, release = n.ps.Join(ctx, topic)
// 	defer release()

// 	return Topic{
// 		Name:  topic,  // TODO:  can we use the client brand/meta for this?
// 		Topic: f.Topic().AddRef(),
// 	}
// }

// func (n *Node) Path() string { return "/" }

// func (n *Node) Ls(ctx context.Context) Iterator {
// 	// TODO(performance):  cache an instance of the View capability
// 	f, release := n.host.View(ctx, nil)
// 	defer release()

// 	it := f.View().Iter(ctx)
// 	runtime.SetFinalizer(it, func(it *cluster.RecordStream) {
// 		it.Finish()
// 	})

// 	return hostSet{
// 		dialer:       dialer(n.vat),
// 		RecordStream: it,
// 	}
// }

// func (n *Node) Walk(ctx context.Context, path string) Anchor {
// 	p := anchor.NewPath(path)
// 	if p.Err() != nil {
// 		return newErrorHost(p.Err())
// 	}

// 	if p.IsRoot() {
// 		return n
// 	}

// 	p, name := p.Next()

// 	id, err := peer.Decode(name)
// 	if err != nil {
// 		return newErrorHost(fmt.Errorf("invalid id: %w", err))
// 	}

// 	return Host{
// 		dialer: dialer(n.vat),
// 		host:   &cluster.Host{Info: peer.AddrInfo{ID: id}},
// 	}.Walk(ctx, p.String())
// }
