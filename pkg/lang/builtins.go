package lang

import (
	"errors"
	"fmt"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

func loadBuiltins(env core.Env, a core.Analyzer) error {
	return bindAll(env,
		comparison(),
		function("nil?", "__isnil__", core.IsNil),
		function("not", "__not__", fnNot),
		function("read", "__read__", fnRead),
		function("render", "__render__", core.Render),
		function("print", "__print__", fnPrint),
		function("len", "__len__", fnLen),
		function("pop", "__pop__", core.Pop),
		function("conj", "__conj__", core.Conj),
		function("type", "__type__", fnTypeOf),
		function("next", "__next__", fnNext))
}

func fnRead(any ww.Any) (core.List, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

func fnNot(any ww.Any) (bool, error) {
	b, err := core.IsTruthy(any)
	return !b, err
}

func fnPrint(any ww.Any) (int, error) {
	s, err := core.Render(any)
	if err != nil {
		return 0, err
	}
	return fmt.Print(s)
}

func fnLen(c core.Countable) (int, error) { return c.Count() }

func fnTypeOf(a ww.Any) (core.Symbol, error) {
	return core.NewSymbol(capnp.SingleSegment(nil), a.MemVal().Which().String())
}

func fnNext(seq core.Seq) (core.Seq, error) { return seq.Next() }

func comparison() bindFunc {
	return func(env core.Env) error {
		return bindAll(env,
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
