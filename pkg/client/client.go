// Package client exports the Wetware client API.
package client

import (
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/debug"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/pubsub"
)

// ErrDisconnected indicates that the client's connection to
// the cluster was lost.
var ErrDisconnected = errors.New("disconnected")

type Node struct {
	casm.Vat

	once sync.Once
	conn *rpc.Conn
	host ww.Host
}

// Bootstrap blocks until the context expires, or the
// node's Host capability resolves.  It is safe to cancel
// the context passed to Dial after this method returns.
func (n *Node) Bootstrap(ctx context.Context) error {
	n.once.Do(func() {
		n.host = ww.Host(n.conn.Bootstrap(ctx))
	})

	return capnp.Client(n.host).Resolve(ctx) // TODO:  remove?
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n *Node) Done() <-chan struct{} {
	return n.conn.Done()
}

// Close the client connection.  Note that this does not
// close the underlying host.
func (n *Node) Close() error {
	return n.conn.Close()
}

func (n *Node) View(ctx context.Context) (cluster.View, capnp.ReleaseFunc) {
	return n.host.View(ctx)
}

// Join a pubsub topic.
func (n *Node) Join(ctx context.Context, topic string) (pubsub.Topic, capnp.ReleaseFunc) {
	router, release := n.host.PubSub(ctx)
	defer release()

	return router.Join(ctx, topic)
}

func (n *Node) Debug(ctx context.Context) (debug.Debugger, capnp.ReleaseFunc) {
	return n.host.Debug(ctx)
}

func (n *Node) Path() string { return "/" }

// func (n *Node) Ls(ctx context.Context) anchor.Iterator {
// 	// // TODO(performance):  cache an instance of the View capability
// 	// f, release := n.host.View(ctx, nil)
// 	// defer release()

// 	// it := f.View().Iter(ctx)
// 	// runtime.SetFinalizer(it, func(it *cluster.RecordStream) {
// 	// 	it.Finish()
// 	// })

// 	// return hostSet{
// 	// 	dialer:       dialer(n.vat),
// 	// 	RecordStream: it,
// 	// }
// }

// func (n *Node) Walk(ctx context.Context, path string) anchor.Anchor {
// 	// p := anchor.NewPath(path)
// 	// if p.Err() != nil {
// 	// 	return newErrorHost(p.Err())
// 	// }

// 	// if p.IsRoot() {
// 	// 	return n
// 	// }

// 	// p, name := p.Next()

// 	// id, err := peer.Decode(name)
// 	// if err != nil {
// 	// 	return newErrorHost(fmt.Errorf("invalid id: %w", err))
// 	// }

// 	// return Host{
// 	// 	dialer: dialer(n.vat),
// 	// 	host:   &cluster.Host{Info: peer.AddrInfo{ID: id}},
// 	// }.Walk(ctx, p.String())
// }
