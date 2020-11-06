// Package lang contains the wetware language iplementation
package lang

import (
	"github.com/spy16/slurp"
	score "github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/builtin"
)

// New returns a new root interpreter.
func New(root ww.Anchor) *slurp.Interpreter {
	if root == nil {
		panic("nil Anchor")
	}

	env := score.New(nil)

	a := builtin.New(root)

	return slurp.New(
		slurp.WithEnv(env),
		slurp.WithAnalyzer(a))
}
