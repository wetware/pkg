// Package lang contains the wetware language iplementation
package lang

import (
	"github.com/spy16/slurp"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/builtin"
	"github.com/wetware/ww/pkg/lang/core"

	_ "github.com/wetware/ww/pkg/lang/builtin/proc" // register default process types
)

// New returns a new root interpreter.
func New(root ww.Anchor) *slurp.Interpreter {
	if root == nil {
		panic("nil Anchor")
	}

	return slurp.New(
		slurp.WithEnv(core.New(nil)),
		slurp.WithAnalyzer(builtin.New(root)))
}
