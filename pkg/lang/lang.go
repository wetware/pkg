// Package lang contains the wetware language iplementation
package lang

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spy16/slurp"
	capnp "zombiezen.com/go/capnproto2"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/lang/reader"
	// _ "github.com/wetware/ww/pkg/lang/core/proc" // register default process types
)

// New returns a new root interpreter.
func New(root ww.Anchor, srcPath ...string) (*slurp.Interpreter, error) {
	if root == nil {
		return nil, errors.New("nil anchor")
	}

	env := core.New()
	if err := bindAll(env, prelude()); err != nil {
		return nil, err
	}

	interp := slurp.New(
		slurp.WithEnv(env),
		slurp.WithAnalyzer(newAnalyzer(root)))

	return interp, source(srcPath).Load(env, interp)
}

func prelude() bindFunc {
	return func(env core.Env) error {
		return bindAll(env,
			function("nil?", "__isnil__", core.IsNil),
			function("not", "__not__", func(any ww.Any) (bool, error) {
				b, err := core.IsTruthy(any)
				return !b, err
			}),
			function("len", "__len__", func(c core.Countable) (int, error) { return c.Count() }),
			function("pop", "__pop__", core.Pop),
			function("conj", "__conj__", core.Conj),
			function("type", "__type__", func(a ww.Any) (core.Symbol, error) {
				return core.NewSymbol(capnp.SingleSegment(nil), a.MemVal().Type().String())
			}),
			function("next", "__next__", func(seq core.Seq) (core.Seq, error) { return seq.Next() }),
			function("=", "__eq__", core.Eq),
			function("<", "__lt__", func(a core.Comparable, b ww.Any) (bool, error) {
				i, err := a.Comp(b)
				return i == -1, err
			}),
			function(">", "__gt__", func(a core.Comparable, b ww.Any) (bool, error) {
				i, err := a.Comp(b)
				return i == 1, err
			}),
			function("<=", "__le__", func(a core.Comparable, b ww.Any) (bool, error) {
				i, err := a.Comp(b)
				return i <= 0, err
			}),
			function(">=", "__ge__", func(a core.Comparable, b ww.Any) (bool, error) {
				i, err := a.Comp(b)
				return i >= 0, err
			}))
	}
}

func function(symbol, name string, fn interface{}) bindFunc {
	return func(env core.Env) error {
		wrapped, err := Func(name, fn)
		if err != nil {
			return err
		}

		return env.Bind(symbol, wrapped)
	}
}

type source []string

func (src source) Load(env core.Env, i *slurp.Interpreter) (err error) {
	var files []os.FileInfo
	for _, path := range src {
		if files, err = ioutil.ReadDir(path); err != nil {
			break
		}

		var paths []string
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".ww") {
				paths = append(paths, filepath.Join(path, f.Name()))
			}
		}

		if err = src.loadFiles(env, i, paths); err != nil {
			break
		}
	}

	return
}

func (source) loadFiles(env core.Env, i *slurp.Interpreter, ps []string) (err error) {
	for _, path := range ps {
		if err = source(nil).loadOne(env, i, path); err != nil {
			break
		}
	}

	// swallow EOF errors
	if errors.Is(err, io.EOF) {
		err = nil
	}

	return
}

func (source) loadOne(env core.Env, i *slurp.Interpreter, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	forms, err := reader.New(f).All()
	if err != nil {
		return err
	}

	for _, form := range forms {
		if _, err = i.Eval(form); err != nil {
			break
		}
	}

	return err
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
