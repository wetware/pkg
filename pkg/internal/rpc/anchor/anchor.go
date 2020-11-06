package anchor

import (
	"context"
	"runtime"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/rpc"
	"github.com/wetware/ww/pkg/lang/builtin"
	"github.com/wetware/ww/pkg/lang/proc"
	"github.com/wetware/ww/pkg/mem"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
	capnp "zombiezen.com/go/capnproto2"
)

type anchor struct {
	path
	anchorProvider
}

func (a anchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return ls(ctx, a.Anchor(), adaptSubanchor(a.path))
}

func (a anchor) Walk(ctx context.Context, path []string) ww.Anchor {
	f, done := a.Anchor().Walk(ctx, func(p api.Anchor_walk_Params) error {
		return p.SetPath(anchorpath.Join(path))
	})

	runtime.SetFinalizer(&f, func(*api.Anchor_walk_Results_Future) {
		done()
	})

	return anchor{
		path:           append(a.path, path...),
		anchorProvider: f,
	}
}

func (a anchor) Load(ctx context.Context) (ww.Any, error) {
	f, done := a.Anchor().Load(ctx, nil)
	defer done()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	v, err := res.Value()
	if err != nil {
		return nil, err
	}

	return builtin.AsAny(mem.Value{Raw: v})
}

func (a anchor) Store(ctx context.Context, any ww.Any) error {
	f, done := a.Anchor().Store(ctx, func(p api.Anchor_store_Params) error {
		return p.SetValue(any.MemVal().Raw)
	})
	defer done()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return ctx.Err()
	}

	if _, err := f.Struct(); err != nil {
		return err
	}

	return nil
}

func (a anchor) Go(ctx context.Context, args ...ww.Any) (ww.Any, error) {
	if len(args) == 0 {
		return nil, errors.New("expected at least one argument, got 0")
	}

	f, done := a.Anchor().Go(ctx, procArgs(args).Set)
	defer done()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	val, err := mem.NewValue(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	if err = val.Raw.SetProc(res.Proc()); err != nil {
		return nil, err
	}

	return proc.FromValue(val), nil
}

type hostAnchor struct {
	id peer.ID
	t  rpc.Terminal
}

// NewHost returns an anchor corresponding to a physical host on the
// network.
func NewHost(t rpc.Terminal, id peer.ID) ww.Anchor {
	return hostAnchor{id: id, t: t}
}

func (h hostAnchor) Name() string   { return h.id.String() }
func (h hostAnchor) Path() []string { return []string{h.Name()} }

func (h hostAnchor) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return h.Walk(ctx, h.Path()).Ls(ctx)
}

func (h hostAnchor) Walk(ctx context.Context, path []string) ww.Anchor {
	return Walk(ctx, h.t, rpc.DialPeer(h.id), path)
}

func (hostAnchor) Load(ctx context.Context) (ww.Any, error) {
	// TODO(enhancement):  return a dict with server info
	return nil, errors.New("hostAnchor.Load NOT IMPLEMENTED")
}

func (hostAnchor) Store(ctx context.Context, any ww.Any) error {
	return errors.New("hostAnchor.Store NOT IMPLEMENTED")
}

func (hostAnchor) Go(context.Context, ...ww.Any) (ww.Any, error) {
	// TODO(enhancement):  run goroutine in the background (i.e. not bound to anchor)
	return nil, errors.New("hostAnchor.Go NOT IMPLEMENTED")
}

type path []string

func (p path) Name() string {
	if anchorpath.Root(nil) {
		return ""
	}

	return p[len(p)-1]
}

func (p path) Path() []string { return p }

type anchorProvider interface {
	Anchor() api.Anchor
}

type procArgs []ww.Any

func (args procArgs) Set(p api.Anchor_go_Params) error {
	vs, err := p.NewArgs(int32(len(args)))
	if err != nil {
		return err
	}

	for i, any := range args {
		if err = vs.Set(i, any.MemVal().Raw); err != nil {
			break
		}
	}

	return err
}
