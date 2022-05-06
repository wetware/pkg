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
	"github.com/wetware/ww/pkg/cap/cluster"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

// ErrDisconnected indicates that the client's connection to
// the cluster was lost.
var ErrDisconnected = errors.New("disconnected")

type Node struct {
	vat  vat.Network
	conn *rpc.Conn
	ps   pscap.PubSub // conn's bootstrap capability
	view cluster.View
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

	return n.view.Client.Resolve(ctx)
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

	return Topic{
		Name:   topic,
		Client: f.Topic().AddRef().Client,
		done:   n.conn.Done(),
	}

	// // Wrap the call to release in a function that ensures
	// // release is only called once.
	// t.release = func() {
	// 	release()
	// 	t.release = nil
	// 	runtime.SetFinalizer(t, nil)
	// }

	// // Ensure finalizer is called if users get sloppy.
	// runtime.SetFinalizer(t, func(t *topicCap) {
	// 	if t.release != nil {
	// 		t.release()
	// 	}
	// })

	// return t
}

func (n Node) Path() []string { return nil }

func (n Node) Ls(ctx context.Context) Iterator {
	s, release := n.view.Iter(ctx)

	it := &hostSet{
		ctx:          ctx,
		dialer:       dialer(n.vat),
		RecordStream: s,
	}

	it.release = func() {
		runtime.SetFinalizer(it, nil)
		release()
	}

	runtime.SetFinalizer(it, func(*hostSet) {
		release()
	})

	return it
}

func (n Node) Walk(ctx context.Context, path []string) Anchor {
	if len(path) == 0 {
		return n
	}

	id, err := peer.Decode(path[0])
	if err != nil {
		return newErrorHost(fmt.Errorf("invalid id: %w", err))
	}

	return Host{
		dialer: dialer(n.vat),
		host:   &cluster.Host{Info: peer.AddrInfo{ID: id}},
	}.Walk(ctx, path[1:])
}
