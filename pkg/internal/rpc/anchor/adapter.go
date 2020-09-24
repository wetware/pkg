package anchor

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/rpc"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

/*
	adapter.go contains utilities for converting Anchor representations from their
	internal API representation to the exported `ww` package representation.
*/

func subanchors(ss api.Anchor_SubAnchor_List, a adapter) (as []ww.Anchor, err error) {
	as = make([]ww.Anchor, ss.Len())
	for i := range as {
		if as[i], err = a.Adapt(ss.At(i)); err != nil {
			break
		}
	}

	return
}

type adapter interface {
	Adapt(api.Anchor_SubAnchor) (ww.Anchor, error)
}

type adaptHostAnchor rpc.Terminal

func (h adaptHostAnchor) Adapt(a api.Anchor_SubAnchor) (ww.Anchor, error) {
	path, err := a.Path()
	if err != nil {
		return nil, err
	}

	parts := anchorpath.Parts(path)

	id, err := peer.Decode(parts[0])
	if err != nil {
		return nil, err
	}

	return NewHost(rpc.Terminal(h), id), nil
}

type adaptSubanchor []string

func (h adaptSubanchor) Adapt(a api.Anchor_SubAnchor) (ww.Anchor, error) {
	subpath, err := a.Path()
	if err != nil {
		return nil, err
	}

	return anchor{
		path:           append(path(h), subpath),
		anchorProvider: a,
	}, nil
}
