package anchor

import (
	"context"

	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

type anchor struct {
	path
	anchorProvider
}

func (a anchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	res, err := a.Anchor().Ls(ctx, func(api.Anchor_ls_Params) error { return nil }).Struct()
	if err != nil {
		return nil, err
	}

	return parseLs(res, anchorLsHandler(a.path))
}

func (a anchor) Walk(ctx context.Context, path []string) ww.Anchor {
	return anchor{
		path: append(a.path, path...),
		anchorProvider: a.Anchor().Walk(ctx, func(p api.Anchor_walk_Params) error {
			return p.SetPath(anchorpath.Join(path))
		}),
	}
}

type hostAnchor struct {
	d rpc.DialString
	t rpc.Terminal
}

func (h hostAnchor) String() string { return string(h.d) }
func (h hostAnchor) Path() []string { return []string{string(h.d)} }

func (h hostAnchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return h.Walk(ctx, h.Path()).Ls(ctx)
}

func (h hostAnchor) Walk(ctx context.Context, path []string) ww.Anchor {
	return Walk(ctx, h.t, h.d, path)
}

type path []string

func (a path) String() string {
	if a == nil || anchorpath.Root(a) {
		return "/"
	}

	return a[len(a)-1]
}

func (a path) Path() []string {
	if a == nil {
		return []string{}
	}

	return a
}

type anchorProvider interface {
	Anchor() api.Anchor
}
