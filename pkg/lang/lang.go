// Package lang contains the wetware language iplementation
package lang

import (
	"errors"

	"github.com/spy16/slurp"
	capnp "zombiezen.com/go/capnproto2"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	// _ "github.com/wetware/ww/pkg/lang/core/proc" // register default process types
)

// New returns a new root interpreter.
func New(root ww.Anchor) (*slurp.Interpreter, error) {
	if root == nil {
		return nil, errors.New("nil anchor")
	}

	env, err := newEnv()
	if err != nil {
		return nil, err
	}

	return slurp.New(
			slurp.WithEnv(env),
			slurp.WithAnalyzer(newAnalyzer(root))),
		nil
}

func newEnv() (core.Env, error) {
	env := core.New()
	return env, bindAll(env,
		prelude(),
		math())
}

func prelude() bindFunc {
	return func(env core.Env) error {
		return bindAll(env,
			function("nil?", "__isnil__", func(any ww.Any) core.Bool {
				return liftBool(core.IsNil(any))
			}),
			function("not", "__not__", func(any ww.Any) (core.Bool, error) {
				b, err := core.IsTruthy(any)
				return liftBool(!b), err
			}),
			function("len", "__len__", func(cnt core.Countable) (core.Int64, error) {
				i, err := cnt.Count()
				if err != nil {
					return nil, err
				}

				return core.NewInt64(capnp.SingleSegment(nil), int64(i))
			}),
			function("pop", "__pop__", core.Pop),
			function("conj", "__conj__", core.Conj))

	}
}

func math() bindFunc {
	return func(env core.Env) error {
		return bindAll(env,
			function("=", "__eq__", func(a, b ww.Any) (core.Bool, error) {
				ok, err := core.Eq(a, b)
				return liftBool(ok), err
			}),
			function("<", "__lt__", func(a core.Comparable, b ww.Any) (core.Bool, error) {
				i, err := a.Comp(b)
				return liftBool(i == -1), err
			}),
			function(">", "__gt__", func(a core.Comparable, b ww.Any) (core.Bool, error) {
				i, err := a.Comp(b)
				return liftBool(i == 1), err
			}),
			function("<=", "__le__", func(a core.Comparable, b ww.Any) (core.Bool, error) {
				i, err := a.Comp(b)
				return liftBool(i <= 0), err
			}),
			function(">=", "__ge__", func(a core.Comparable, b ww.Any) (core.Bool, error) {
				i, err := a.Comp(b)
				return liftBool(i >= 0), err
			}))
	}
}

type bindable interface {
	Bind(core.Env) error
}

func bindAll(env core.Env, bs ...bindable) (err error) {
	for _, b := range bs {
		if err = b.Bind(env); err != nil {
			break
		}
	}

	return
}

type bindFunc func(core.Env) error

func (bind bindFunc) Bind(env core.Env) error { return bind(env) }

func function(symbol, name string, fn interface{}) bindFunc {
	return func(env core.Env) error {
		wrapped, err := core.Func(name, fn)
		if err != nil {
			return err
		}

		return env.Bind(symbol, wrapped)
	}
}

func liftBool(b bool) core.Bool {
	if b {
		return core.True
	}

	return core.False
}
