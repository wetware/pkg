package client

import (
	"context"
	"fmt"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/wetware/ww/pkg/cap/anchor"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

type Iterator interface {
	Err() error
	Next() (more bool)
	Anchor() Anchor
}

type Anchor interface {
	Path() string
	Ls(ctx context.Context) Iterator
	Walk(ctx context.Context, path string) Anchor
}

type Container interface {
	Set(ctx context.Context, data []byte) error
	Get(ctx context.Context) (data []byte, release func())
}

type dialer vat.Network

func (d dialer) Dial(ctx context.Context, info peer.AddrInfo) (*rpc.Conn, error) {
	return vat.Network(d).Connect(ctx, info, cluster.HostCapability)
}

// Host anchor represents a machine instance.
type Host struct {
	dialer dialer
	host   *cluster.Host
}

func newErrorHost(err error) Host {
	return Host{host: &cluster.Host{
		Client: capnp.ErrorClient(err),
	}}
}

func (h Host) ID() peer.ID           { return h.host.Info.ID }
func (h Host) Addrs() []ma.Multiaddr { return h.host.Info.Addrs }

func (h Host) Path() string {
	return fmt.Sprintf("/%s", h.host.Info.ID)
}

func (h Host) Ls(ctx context.Context) Iterator {
	rs, release := h.host.Ls(ctx, h.dialer)

	it := &iterator{
		Iterator: rs,
		path:     anchor.NewPath(h.Path()),
	}

	it.release = func() {
		runtime.SetFinalizer(it, nil)
		release()
	}

	runtime.SetFinalizer(it, func(*iterator) {
		release()
	})

	return it
}

func (h Host) Walk(ctx context.Context, path string) Anchor {
	p := anchor.NewPath(path)
	if err := p.Err(); err != nil {
		return newErrorHost(fmt.Errorf("path: %w", err))
	}

	if p.IsRoot() {
		return h
	}

	r, release := h.host.Walk(ctx, h.dialer, p)
	runtime.SetFinalizer(&r, func(*anchor.Anchor) {
		release()
	})

	return anchorClient{
		path:   p,
		Anchor: r,
	}
}

type hostSet struct {
	dialer dialer
	*cluster.RecordStream
}

func (hs hostSet) Err() error { return hs.RecordStream.Err }

func (hs hostSet) Next() bool {
	hs.RecordStream.Next()
	return hs.More
}

func (hs hostSet) Anchor() Anchor {
	return Host{
		dialer: hs.dialer,
		host: &cluster.Host{
			Info: peer.AddrInfo{
				ID: hs.RecordStream.Record().Peer(),
			},
		},
	}
}

type iterator struct {
	*anchor.Iterator
	release capnp.ReleaseFunc
	path    anchor.Path
}

func (it *iterator) Err() error { return it.Iterator.Err }

func (it *iterator) Anchor() Anchor {
	return anchorClient{
		path:   it.path.WithChild(it.Name),
		Anchor: it.Iterator.Anchor().AddRef(),
	}
}

type anchorClient struct {
	path anchor.Path
	anchor.Anchor
}

func (r anchorClient) Path() string { return r.path.String() }

func (r anchorClient) Ls(ctx context.Context) Iterator {
	rs, release := r.Anchor.Ls(ctx)

	it := &iterator{
		path:     r.path,
		Iterator: rs,
	}

	it.release = func() {
		runtime.SetFinalizer(it, nil)
		release()
	}

	runtime.SetFinalizer(it, func(*iterator) {
		release()
	})

	return it
}

func (r anchorClient) Walk(ctx context.Context, path string) Anchor {
	if len(path) == 0 {
		return r
	}

	var release capnp.ReleaseFunc
	r.Anchor, release = r.Anchor.Walk(ctx, anchor.NewPath(path))
	runtime.SetFinalizer(&r, func(*anchorClient) {
		release()
	})

	return &r
}
