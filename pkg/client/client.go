// Package client exports the Wetware client API.
package client

import (
	"context"

	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

type Node struct {
	vat  vat.Network
	conn *rpc.Conn
	ps   pscap.PubSub // conn's bootstrap capability
}

// String returns the cluster namespace
func (n Node) String() string { return n.vat.NS }

func (n Node) Loggable() map[string]interface{} {
	return n.vat.Loggable()
}

// Bootstrap blocks until the context expires, or the
// node's capabilities resolve.  It is safe to cancel
// the context passed to Dial after this method returns.
func (n Node) Bootstrap(ctx context.Context) error {
	// TODO:  update this when we replace 'ps' with a
	//        capability set.
	return n.ps.Client.Resolve(ctx)
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n Node) Done() <-chan struct{} {
	return n.conn.Done()
}

func (n Node) Close() error {
	n.ps.Release()

	return n.conn.Close()
}

func (n Node) Join(ctx context.Context, topic string) *Topic {
	var f, release = n.ps.Join(ctx, topic)

	t := &Topic{f: f}
	t.Release = func() {
		release()
		t.Release = nil
	}

	// Ensure finalizer is called if users get sloppy.
	runtime.SetFinalizer(t, func(t *Topic) {
		if t.Release != nil {
			t.Release()
		}
	})

	return t
}
