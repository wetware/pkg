package anchor

import (
	"context"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p-core/peer"
	api "github.com/wetware/ww/internal/api/cluster"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/vat"
)

var Capability = vat.BasicCap{
	"hostAnchor/packed",
	"hostAnchor"}

type HostAnchor struct {
	Peer peer.ID
	Vat  vat.Network

	client api.Host

	once sync.Once
}

func (ha HostAnchor) Ls(ctx context.Context, path []string) (AnchorIterator, error) {
	if err := ha.bootstrapOnce(ctx); err != nil {
		return nil, err
	}

	return nil, nil // TODO
}

func (ha HostAnchor) Walk(ctx context.Context, path []string) (Anchor, error) {
	if err := ha.bootstrapOnce(ctx); err != nil {
		return nil, err
	}

	fut, release := ha.client.Walk(ctx, func(a api.Anchor_walk_Params) error {
		capPath, err := a.NewPath(int32(len(path)))
		if err != nil {
			return err
		}
		for i, e := range path {
			if err := capPath.Set(i, e); err != nil {
				return err
			}
		}
		return nil
	})

	return ContainerAnchor{fut: fut, release: release}, nil
}

func (ha HostAnchor) bootstrapOnce(ctx context.Context) error {
	var (
		conn *rpc.Conn
		err  error
	)

	ha.once.Do(func() {
		conn, err = ha.Vat.Connect(
			ctx,
			peer.AddrInfo{ID: ha.Peer},
			Capability,
		)
		if err != nil {
			return
		}
		ha.client = api.Host{Client: conn.Bootstrap(ctx)}
	})

	return err
}

type HostAnchorIterator struct {
	Vat     vat.Network
	It      cluster.Iterator
	Release capnp.ReleaseFunc
}

func (hai HostAnchorIterator) Next(ctx context.Context) error {
	hai.It.Next(ctx)
	return hai.It.Err
}

func (hai HostAnchorIterator) Finish() {
	hai.Release()
}

func (hai HostAnchorIterator) Anchor() Anchor {
	return HostAnchor{
		Peer: hai.It.Record().Peer(),
		Vat:  hai.Vat,
	}
}
