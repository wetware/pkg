// Package client exports the Wetware client API.
package client

import (
	"context"
	"errors"

	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/cap/cluster/anchor"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

var (
	ErrInvalidPath = errors.New("invalid path")
)

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

func (n Node) Ls(ctx context.Context, path []string) (anchor.AnchorIterator, error) {
	if len(path) == 0 {
		// TODO: return View iterator wrapped in Anchor transformer
	}

	a, err := n.hostAnchor(ctx, path)
	if err != nil {
		return nil, err
	}

	return a.Ls(ctx, path[1:])

}

func (n Node) Walk(ctx context.Context, path []string) (anchor.Anchor, error) {
	if len(path) == 0 {
		// TODO: return a RootAnchor that is equivalent to this same implementation
		return nil, ErrInvalidPath
	}

	a, err := n.hostAnchor(ctx, path)
	if err != nil {
		return nil, err
	}

	if len(path) > 1 {
		return a.Walk(ctx, path[1:])
	} else {
		return a, nil
	}

}

func (n Node) hostAnchor(ctx context.Context, path []string) (anchor.Anchor, error) {
	fut, release := n.view.Lookup(ctx, peer.ID(path[0]))
	defer release()

	select {
	case <-fut.Done():
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	rec, err := fut.Struct()
	if err != nil {
		return nil, err
	}

	peer, err := rec.Peer()
	if err != nil {
		return nil, err
	}

	return anchor.HostAnchor{
		Peer: peer,
		Vat:  n.vat,
	}, nil
}
