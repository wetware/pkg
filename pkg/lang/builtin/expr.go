package builtin

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/spy16/slurp/builtin"
	score "github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ core.Expr = (*ConstExpr)(nil)
	_ core.Expr = (*IfExpr)(nil)
	_ core.Expr = (*ResolveExpr)(nil)
	_ core.Expr = (*DefExpr)(nil)
	_ core.Expr = (*PathExpr)(nil)
	_ core.Expr = (*LocalGoExpr)(nil)
	_ core.Expr = (*RemoteGoExpr)(nil)
	_ core.Expr = (*InvokeExpr)(nil)
	// _ core.Expr = (*)(nil)

	_ core.Invokable = (*PathExpr)(nil)
)

type (
	// DoExpr represents the (do expr*) form.
	DoExpr = builtin.DoExpr

	// QuoteExpr expression represents a quoted form
	QuoteExpr = builtin.QuoteExpr
)

// ConstExpr returns the Const value wrapped inside when evaluated. It has
// no side-effect on the VM.
type ConstExpr struct{ form ww.Any }

// Eval returns the constant value unmodified.
func (ce ConstExpr) Eval(_ core.Env) (score.Any, error) { return ce.form, nil }

// IfExpr represents the if-then-else form.
type IfExpr struct{ Test, Then, Else core.Expr }

// Eval evaluates the then or else expr based on truthiness of the test
// expr result.
func (ife IfExpr) Eval(env core.Env) (score.Any, error) {
	target := ife.Else
	if ife.Test != nil {
		test, err := ife.Test.Eval(env)
		if err != nil {
			return nil, err
		}

		ok, err := core.IsTruthy(test.(ww.Any))
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
	return target.Eval(env)
}

// ResolveExpr resolves a symbol from the given environment.
type ResolveExpr struct{ Symbol Symbol }

// Eval resolves the symbol in the given environment or its parent env
// and returns the result. Returns ErrNotFound if the symbol was not
// found in the entire hierarchy.
func (re ResolveExpr) Eval(env core.Env) (v score.Any, err error) {
	var sym string
	if sym, err = re.Symbol.Raw.Symbol(); err != nil {
		return
	}

	for env != nil {
		if v, err = env.Resolve(sym); errors.Is(err, core.ErrNotFound) {
			// not found in the current frame. check parent.
			env = env.Parent()
			continue
		}

		// found the symbol or there was some unexpected error.
		break
	}

	return
}

// DefExpr represents the (def name value) binding form.
type DefExpr struct {
	Name  string
	Value core.Expr
}

// Eval creates the binding with the name and value in Root env.
func (de DefExpr) Eval(env core.Env) (score.Any, error) {
	var val score.Any
	var err error
	if de.Value != nil {
		val, err = de.Value.Eval(env)
		if err != nil {
			return nil, err
		}
	} else {
		val = Nil{}
	}

	if err := score.Root(env).Bind(de.Name, val); err != nil {
		return nil, err
	}

	return NewSymbol(capnp.SingleSegment(nil), de.Name)
}

// InvokeExpr performs invocation of target when evaluated.
type InvokeExpr struct {
	Name   string
	Target core.Expr
	Args   []core.Expr
}

// Eval evaluates the target expr and invokes the result if it is an
// Invokable  Returns error otherwise.
func (ie InvokeExpr) Eval(env core.Env) (score.Any, error) {
	val, err := ie.Target.Eval(env)
	if err != nil {
		return nil, err
	}

	fn, ok := val.(core.Invokable)
	if !ok {
		return nil, core.Error{
			Cause:   core.ErrNotInvokable,
			Message: fmt.Sprintf("value of type '%s' is not invokable", reflect.TypeOf(val)),
		}
	}

	var any score.Any
	args := make([]ww.Any, len(ie.Args))
	for i, ae := range ie.Args {
		if any, err = ae.Eval(env); err != nil {
			return nil, err
		}

		args[i] = any.(ww.Any) // TODO(performance): can we use unsafe.Pointer here?
	}

	return fn.Invoke(args...)
}

// PathExpr binds a path to an Anchor
type PathExpr struct {
	Root ww.Anchor
	Path
}

// Eval returns the PathExpr unmodified
func (pex PathExpr) Eval(core.Env) (score.Any, error) { return pex, nil }

// Invoke is the data selector for the Path type.  It gets/sets the value at the anchor
// path.
func (pex PathExpr) Invoke(args ...ww.Any) (ww.Any, error) {
	path, err := pex.Parts()
	if err != nil {
		return nil, err
	}

	anchor := pex.Root.Walk(context.Background(), path)

	if len(args) == 0 {
		return anchor.Load(context.Background())
	}

	err = anchor.Store(context.Background(), args[0])
	if err != nil {
		return nil, core.Error{
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
func (plx PathListExpr) Eval(core.Env) (score.Any, error) {
	path, err := plx.Path.Parts()
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
	Args []ww.Any
}

// Eval resolves starts the process.
func (lx LocalGoExpr) Eval(env core.Env) (score.Any, error) {
	return core.Spawn(env.Child("<goroutine>", nil), lx.Args...)
}

// RemoteGoExpr starts a global process.  Global processes may be bound to an Anchor,
// rendering them addressable by remote hosts.
type RemoteGoExpr struct {
	Root ww.Anchor
	Path Path
	Args []ww.Any
}

// Eval resolves the anchor and starts the process.
func (rx RemoteGoExpr) Eval(core.Env) (score.Any, error) {
	path, err := rx.Path.Parts()
	if err != nil {
		return nil, err
	}

	return rx.Root.Walk(context.Background(), path).
		Go(context.Background(), rx.Args...)
}
