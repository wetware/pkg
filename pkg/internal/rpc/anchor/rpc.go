package anchor

import (
	"context"
	"runtime"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/rpc"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

// Ls .
func Ls(ctx context.Context, t rpc.Terminal, d rpc.Dialer) ([]ww.Anchor, error) {
	c := t.Dial(ctx, d, ww.AnchorProtocol)
	defer t.HangUp(c)

	return ls(ctx, api.Anchor{Client: c.Client}, adaptHostAnchor(t))
}

func ls(ctx context.Context, a api.Anchor, ad adapter) ([]ww.Anchor, error) {
	f, done := a.Ls(ctx, func(api.Anchor_ls_Params) error { return nil })
	defer done()

	select {
	case <-f.Done(): // promise has resolved
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	res, err := f.Struct()
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
	c := t.Dial(ctx, d, ww.AnchorProtocol)
	defer t.HangUp(c)

	return walk(ctx, api.Anchor{Client: c.Client}, path)
}

func walk(ctx context.Context, a api.Anchor, p path) ww.Anchor {
	f, done := a.Walk(ctx, func(ps api.Anchor_walk_Params) error {
		return ps.SetPath(anchorpath.Join(p.Path()))
	})

	runtime.SetFinalizer(&f, func(*api.Anchor_walk_Results_Future) {
		done()
	})

	return anchor{
		path:           p,
		anchorProvider: f,
	}
}
