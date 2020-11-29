// Package lang contains the wetware language iplementation
package lang

import (
	"errors"

	"github.com/spy16/slurp"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/builtin"
	"github.com/wetware/ww/pkg/lang/core"

	_ "github.com/wetware/ww/pkg/lang/builtin/proc" // register default process types
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
			slurp.WithAnalyzer(builtin.New(root))),
		nil
}

func newEnv() (core.Env, error) {
	env := core.New()
	return env, bindAll(env,
		prelude)
}

func bindAll(env core.Env, bs ...func(core.Env) error) (err error) {
	for _, bind := range bs {
		if err = bind(env); err != nil {
			break
		}
	}

	return
}

func prelude(env core.Env) error {
	v, err := builtin.Func("__eq__", func(a, b ww.Any) (core.Bool, error) {
		ok, err := core.Eq(a, b)
		if ok {
			return builtin.True, err
		}

		return builtin.False, err
	})

	if err != nil {
		return err
	}

	return env.Bind("=", v)
}
