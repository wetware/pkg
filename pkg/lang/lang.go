// Package lang contains the wetware language iplementation
package lang

import (
	"errors"
	"fmt"

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

	env, err := core.New()
	if err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}

	return slurp.New(
			slurp.WithEnv(env),
			slurp.WithAnalyzer(builtin.New(root))),
		nil
}
