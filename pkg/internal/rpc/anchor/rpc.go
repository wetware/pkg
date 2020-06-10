package anchor

import (
	"context"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
)

// Ls .
func Ls(ctx context.Context, t rpc.Terminal, d rpc.Dialer) ([]ww.Anchor, error) {
	c := t.Dial(ctx, d, ww.Protocol)
	defer t.HangUp(c)

	return ls(ctx, api.Anchor{Client: c}, adaptHostAnchor(t))
}

func ls(ctx context.Context, a api.Anchor, ad adapter) ([]ww.Anchor, error) {
	res, err := a.Ls(ctx, func(api.Anchor_ls_Params) error { return nil }).Struct()
	if err != nil {
		return nil, err
	}

	cs, err := res.Children()
	if err != nil {
		return nil, err
	}

	return subanchors(cs, ad)
}

// Walk returns the anchor at the specified path.
func Walk(ctx context.Context, t rpc.Terminal, d rpc.Dialer, path []string) ww.Anchor {
	c := t.Dial(ctx, d, ww.Protocol)
	defer t.HangUp(c)

	return walk(ctx, api.Anchor{Client: c}, path)
}

func walk(ctx context.Context, a api.Anchor, p path) ww.Anchor {
	return anchor{
		path: p,
		anchorProvider: a.Walk(ctx, func(ps api.Anchor_walk_Params) error {
			return ps.SetPath(p.Absolute())
		}),
	}
}
