package cluster

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/vat"
)

type rootAnchor struct {
	vat  vat.Network
	view *View
}

func NewRootAnchor(vat vat.Network, view *View) Anchor {
	return rootAnchor{vat: vat, view: view}
}

func (r rootAnchor) Name() string {
	return "/"
}

func (r rootAnchor) Path() []string {
	return []string{}
}

func (r rootAnchor) Ls(ctx context.Context) (AnchorIterator, error) {
	it, release := r.view.Iter(ctx)
	return hostAnchorIterator{vat: r.vat, it: it, release: release}, nil
}

func (r rootAnchor) Walk(ctx context.Context, path []string) (Anchor, error) {
	if len(path) == 0 {
		return r, nil
	}

	id, err := peer.Decode(path[0])
	if err != nil {
		return nil, err
	}
	host := HostAnchor{
		Peer: id,
		vat:  r.vat,
	}

	return host.Walk(ctx, path[1:])
}
