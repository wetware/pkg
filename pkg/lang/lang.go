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

	env := core.New()

	err := bindAll(env,
		prelude)

	return slurp.New(
			slurp.WithEnv(env),
			slurp.WithAnalyzer(builtin.New(root))),
		err
}

type module func(core.Env) error

func bindAll(env core.Env, mods ...module) (err error) {
	for _, bindModule := range mods {
		if err = bindModule(env); err != nil {
			break
		}
	}

	return
}

func prelude(env core.Env) (err error) {
	return errors.New("prelude not implemented")
	// for _, bind := range []struct {
	// 	name  string
	// 	value ww.Any
	// }{
	// 	// {
	// 	// 	// ...
	// 	// },
	// } {
	// 	if err = env.Bind(bind.name, bind.value); err != nil {
	// 		break
	// 	}
	// }

	// return
}
