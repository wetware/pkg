package anchor

import (
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
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

	return hostAnchor{
		d: rpc.DialString(path),
		t: rpc.Terminal(h),
	}, nil
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
