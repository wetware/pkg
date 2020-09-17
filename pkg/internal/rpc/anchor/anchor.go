package anchor

import (
	"context"
	"errors"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/rpc"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

type anchor struct {
	path
	anchorProvider
}

func (a anchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return ls(ctx, a.Anchor(), adaptSubanchor(a.path))
}

func (a anchor) Walk(ctx context.Context, path []string) ww.Anchor {
	return anchor{
		path: append(a.path, path...),
		anchorProvider: a.Anchor().Walk(ctx, func(p api.Anchor_walk_Params) error {
			return p.SetPath(anchorpath.Join(path))
		}),
	}
}

func (a anchor) Load(ctx context.Context) (api.Value, error) {
	res, err := a.Anchor().Load(ctx, func(api.Anchor_load_Params) error { return nil }).Struct()
	if err != nil {
		return api.Value{}, err
	}

	return res.Value()
}

func (a anchor) Store(ctx context.Context, v api.Value) error {
	if _, err := a.Anchor().Store(ctx, func(p api.Anchor_store_Params) error {
		return p.SetValue(v)
	}).Struct(); err != nil {
		return err
	}

	return nil
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

func (hostAnchor) Load(ctx context.Context) (api.Value, error) {
	// TODO(enhancement):  return a dict with server info
	return api.Value{}, errors.New("hostAnchor.Load NOT IMPLEMENTED")
}

func (hostAnchor) Store(ctx context.Context, v api.Value) error {
	return errors.New("hostAnchor.Store NOT IMPLEMENTED")
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

func (a path) Absolute() string {
	return anchorpath.Join(a)
}

type anchorProvider interface {
	Anchor() api.Anchor
}
