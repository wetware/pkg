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
func (pe PathExpr) Eval(env *parens.Env) (parens.Any, error) {
	return pe, nil
}

// Invoke is the data selector for the Path type.  It gets/sets the value at the anchor
// path.
func (pe PathExpr) Invoke(_ *parens.Env, args ...parens.Any) (parens.Any, error) {
	path, err := pe.Parts()
	if err != nil {
		return nil, err
	}

	anchor := pe.Root.Walk(context.Background(), path)

	if len(args) == 0 {
		v, err := anchor.Load(context.Background())
		if err != nil {
			return nil, parens.Error{
				Cause:   err,
				Message: anchorpath.Join(path),
			}
		}

		return valueOf(v)
	}

	err = anchor.Store(context.Background(), args[0].(apiValueProvider).Value())
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
func (le PathListExpr) Eval(_ *parens.Env) (parens.Any, error) {
	path, err := le.Parts()
	if err != nil {
		return nil, err
	}

	as, err := le.Root.Walk(context.Background(), path).Ls(context.Background())
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

// // GoExpr evaluates an expression in a separate goroutine.
// type GoExpr struct {
// 	Value parens.Any
// }

// // Eval forks the given context to get a child context and launches goroutine
// // with the child context to evaluate the
// func (ge GoExpr) Eval(env *parens.Env) (parens.Any, error) {
// 	child := env.Fork()
// 	go func() {
// 		_, _ = child.Eval(ge.Value)
// 	}()
// 	return nil, nil
// }
