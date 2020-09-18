package lang

import (
	"context"

	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ parens.Expr = (*PathExpr)(nil)
	_ parens.Expr = (*PathListExpr)(nil)

	_ parens.Invokable = (*PathExpr)(nil)
)

// PathExpr binds a path to an Anchor
type PathExpr struct {
	Root ww.Anchor
	Path
}

// Eval returns the PathExpr unmodified
func (pex PathExpr) Eval(env *parens.Env) (parens.Any, error) {
	return pex, nil
}

// Invoke is the data selector for the Path type.  It gets/sets the value at the anchor
// path.
func (pex PathExpr) Invoke(_ *parens.Env, args ...parens.Any) (parens.Any, error) {
	path, err := pex.Parts()
	if err != nil {
		return nil, err
	}

	anchor := pex.Root.Walk(context.Background(), path)

	if len(args) == 0 {
		return anchor.Load(context.Background())
	}

	err = anchor.Store(context.Background(), args[0].(ww.Any))
	if err != nil {
		return nil, parens.Error{
			Cause:   err,
			Message: anchorpath.Join(path),
		}
	}

	return nil, nil
}

// PathListExpr fetches subanchors for a path
type PathListExpr struct{ PathExpr }

// Eval calls ww.Anchor.Ls
func (plx PathListExpr) Eval(_ *parens.Env) (parens.Any, error) {
	path, err := plx.Parts()
	if err != nil {
		return nil, err
	}

	as, err := plx.Root.Walk(context.Background(), path).Ls(context.Background())
	if err != nil {
		return nil, err
	}

	b, err := NewVectorBuilder(capnp.SingleSegment(nil))
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		// TODO(performance):  cache the anchors.
		p, err := NewPath(capnp.SingleSegment(nil), a.String())
		if err != nil {
			return nil, err
		}

		if err = b.Conj(p); err != nil {
			return nil, err
		}
	}

	return b.Vector()
}

// RemoteProcExpr starts a remote goroutine.
type RemoteProcExpr struct {
	PathExpr
	Spec ww.ProcSpec
}

// Eval resolves the anchor and starts the remote goroutine.
func (rpx RemoteProcExpr) Eval(env *parens.Env) (parens.Any, error) {
	path, err := rpx.Parts()
	if err != nil {
		return nil, err
	}

	return nil, rpx.Root.Walk(context.Background(), path).
		Go(context.Background(), rpx.Spec)
}
