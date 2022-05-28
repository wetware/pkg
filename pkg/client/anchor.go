package client

import (
	"context"
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
	Path() []string
	Ls(ctx context.Context) Iterator
	Walk(ctx context.Context, path []string) Anchor
}

type Container interface {
	Set(ctx context.Context, data []byte) error
	Get(ctx context.Context) (data []byte, release func())
}

type dialer vat.Network

func (d dialer) Dial(ctx context.Context, info peer.AddrInfo) (*rpc.Conn, error) {
	return vat.Network(d).Connect(ctx, info, anchor.AnchorCapability)
}

// Host anchor represents a machine instance.
type Host struct {
	dialer dialer
	host   *anchor.Host
}

func newErrorHost(err error) Host {
	return Host{host: &anchor.Host{
		Client: capnp.ErrorClient(err),
	}}
}

func (h Host) ID() peer.ID           { return h.host.Info.ID }
func (h Host) Addrs() []ma.Multiaddr { return h.host.Info.Addrs }

func (h Host) Path() []string {
	return []string{h.host.Info.ID.String()}
}

func (h Host) Join(ctx context.Context, peers ...peer.AddrInfo) error {
	return h.host.Join(ctx, h.dialer, peers)
}

func (h Host) Ls(ctx context.Context) Iterator {
	rs, release := h.host.Ls(ctx, h.dialer)

	it := &registerMap{
		RegisterMap: rs,
		path:        h.Path(),
	}

	it.release = func() {
		runtime.SetFinalizer(it, nil)
		release()
	}

	runtime.SetFinalizer(it, func(*registerMap) {
		release()
	})

	return it
}

func (h Host) Walk(ctx context.Context, path []string) Anchor {
	if len(path) == 0 {
		return h
	}

	r, release := h.host.Walk(ctx, h.dialer, path)
	runtime.SetFinalizer(&r, func(*anchor.Register) {
		release()
	})

	return register{
		path:     path,
		Register: r,
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
		host: &anchor.Host{
			Info: peer.AddrInfo{
				ID: hs.RecordStream.Record().Peer(),
			},
		},
	}
}

type registerMap struct {
	*anchor.RegisterMap
	release capnp.ReleaseFunc
	path    []string
}

func (it *registerMap) Err() error { return it.RegisterMap.Err }

func (it *registerMap) Anchor() Anchor {
	return register{
		path:     append(it.path, it.Name),
		Register: it.Register().AddRef(),
	}
}

type register struct {
	path []string
	anchor.Register
}

func (r register) Path() []string { return r.path }

func (r register) Ls(ctx context.Context) Iterator {
	rs, release := r.Register.Ls(ctx)

	it := &registerMap{
		path:        r.Path(),
		RegisterMap: rs,
	}

	it.release = func() {
		runtime.SetFinalizer(it, nil)
		release()
	}

	runtime.SetFinalizer(it, func(*registerMap) {
		release()
	})

	return it
}

func (r register) Walk(ctx context.Context, path []string) Anchor {
	if len(path) == 0 {
		return r
	}

	var release capnp.ReleaseFunc
	r.Register, release = r.Register.Walk(ctx, path)
	runtime.SetFinalizer(&r, func(*register) {
		release()
	})

	return &r
}
