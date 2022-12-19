// Package client exports the Wetware client API.
package client

import (
	"context"

	"capnproto.org/go/capnp/v3"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/debug"
	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/pubsub"
)

type Node struct {
	Vat casm.Vat
	Conn
}

// Loggable fields for the node.
func (n *Node) Loggable() map[string]any {
	return n.Vat.Loggable()
}

// Host to which the client node is connected.  This is a low-level API
// that is not subject to Wetware's backwards-compatibility guarantees.
// Users are encouraged to access this functionality by calling Node's
// other methods.
func (n *Node) Host(ctx context.Context) host.Host {
	client := n.Conn.Bootstrap(ctx)
	return host.Host(client)
}

// Bootstrap blocks until the bootstrap capability has been resolved.
func (n *Node) Bootstrap(ctx context.Context) error {
	return capnp.Client(n.Host(ctx)).Resolve(ctx)
}

func (n *Node) View(ctx context.Context) (cluster.View, capnp.ReleaseFunc) {
	return n.Host(ctx).View(ctx)
}

// Join a pubsub topic.
func (n *Node) Join(ctx context.Context, topic string) (pubsub.Topic, capnp.ReleaseFunc) {
	router, release := n.Host(ctx).PubSub(ctx)
	defer release()

	return router.Join(ctx, topic)
}

func (n *Node) Debug(ctx context.Context) (debug.Debugger, capnp.ReleaseFunc) {
	return n.Host(ctx).Debug(ctx)
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
