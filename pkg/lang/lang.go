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
func New(root ww.Anchor, srcPath ...string) (*slurp.Interpreter, error) {
	if root == nil {
		return nil, errors.New("nil anchor")
	}

	env := core.New()

	a, err := newAnalyzer(root, srcPath)
	if err != nil {
		return nil, err
	}

	return slurp.New(
			slurp.WithEnv(env),
			slurp.WithAnalyzer(a)),
		prelude(env, a)
}

func prelude(env core.Env, a core.Analyzer) (err error) {
	if err = loadBuiltins(env, a); err != nil {
		return
	}

	return loadPrelude(env, a)
}

func loadPrelude(env core.Env, a core.Analyzer) error {
	// We're effectively running `(import :prelude)`

	sym, err := core.NewSymbol(capnp.SingleSegment(nil), "import")
	if err != nil {
		return err
	}

	kw, err := core.NewKeyword(capnp.SingleSegment(nil), "prelude")
	if err != nil {
		return err
	}

	// (import :prelude)
	form, err := core.NewList(capnp.SingleSegment(nil), sym, kw)
	if err != nil {
		return err
	}

	_, err = core.Eval(env, a, form)
	return err
}
