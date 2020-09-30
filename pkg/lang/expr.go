package lang

import (
	"context"

	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/proc"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ parens.Expr = (*PathExpr)(nil)
	_ parens.Expr = (*PathListExpr)(nil)

	_ parens.Invokable = (*PathExpr)(nil)
)

// IfExpr represents the if-then-else form.
type IfExpr struct{ Test, Then, Else parens.Expr }

// Eval the expression
func (ife IfExpr) Eval() (parens.Any, error) {
	var target = ife.Else
	if ife.Test != nil {
		test, err := ife.Test.Eval()
		if err != nil {
			return nil, err
		}

		ok, err := IsTruthy(test.(ww.Any))
		if err != nil {
			return nil, err
		}

		if ok {
			target = ife.Then
		}
	}

	if target == nil {
		return Nil{}, nil
	}

	return target.Eval()
}

// PathExpr binds a path to an Anchor
type PathExpr struct {
	Root ww.Anchor
	Path
}

// Eval returns the PathExpr unmodified
func (pex PathExpr) Eval() (parens.Any, error) { return pex, nil }

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
type PathListExpr struct {
	PathExpr
	Args []ww.Any
}

// Eval calls ww.Anchor.Ls
func (plx PathListExpr) Eval() (parens.Any, error) {
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
		p, err := NewPath(capnp.SingleSegment(nil), anchorpath.Join(a.Path()))
		if err != nil {
			return nil, err
		}

		if err = b.Conj(p); err != nil {
			return nil, err
		}
	}

	return b.Vector()
}

// LocalGoExpr starts a local process.  Local processes cannot be addressed by remote
// hosts.
type LocalGoExpr struct {
	Env  *parens.Env
	Args []ww.Any
}

// Eval resolves starts the process.
func (lx LocalGoExpr) Eval() (parens.Any, error) {
	return proc.Spawn(lx.Env.Fork(), lx.Args...)
}

// GlobalGoExpr starts a global process.  Global processes may be bound to an Anchor,
// rendering them addressable by remote hosts.
type GlobalGoExpr struct {
	Root ww.Anchor
	Path Path
	Args []ww.Any
}

// Eval resolves the anchor and starts the process.
func (gx GlobalGoExpr) Eval() (parens.Any, error) {
	path, err := gx.Path.Parts()
	if err != nil {
		return nil, err
	}

	return gx.Root.Walk(context.Background(), path).Go(context.Background(), gx.Args...)
}
