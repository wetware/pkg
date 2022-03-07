// Package client exports the Wetware client API.
package client

import (
	"context"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/wetware/ww/pkg/cap/cluster"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

type Node struct {
	vat   vat.Network
	conns []*rpc.Conn
	ps    pscap.PubSub // conn's bootstrap capability
	view  cluster.View
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
	if err := n.ps.Client.Resolve(ctx); err != nil {
		return err
	}
	return n.view.Client.Resolve(ctx)
}

// Done returns a read-only channel that is closed when
// 'n' becomes disconnected from the cluster.
func (n Node) Done() <-chan struct{} {
	return n.conns[0].Done()
}

func (n Node) Close() error {
	n.ps.Release()

	var err error

	for _, conn := range n.conns {
		if tmpErr := conn.Close(); tmpErr != nil {
			err = tmpErr
		}
	}
	return err
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

func (n Node) Ls(ctx context.Context) (cluster.AnchorIterator, error) {
	return cluster.NewRootAnchor(n.vat, &n.view).Ls(ctx)
}

func (n Node) Walk(ctx context.Context, path []string) (cluster.Anchor, error) {
	return cluster.NewRootAnchor(n.vat, &n.view).Walk(ctx, path)
}
