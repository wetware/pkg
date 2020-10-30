package lang

import (
	"errors"
	"fmt"

	"github.com/spy16/slurp/builtin"
	"github.com/spy16/slurp/core"
	ww "github.com/wetware/ww/pkg"
)

var (
	_ core.Analyzer = formAnalyzer{}
)

type formAnalyzer struct {
	root         ww.Anchor
	specialForms map[string]builtin.ParseSpecial
}

func analyzer(root ww.Anchor) formAnalyzer {
	// c := anchorClient{root}

	return formAnalyzer{
		root:         root,
		specialForms: map[string]builtin.ParseSpecial{
			// "go":    c.Go,
			// "do":    parseDoExpr,
			// "if":    parseIfExpr,
			// "def":   parseDefExpr,
			// "quote": parseQuoteExpr,
			// "ls":    c.Ls,
			// "pop":   parsePop,
			// "conj":  parseConj,
		},
	}
}

// Analyze performs syntactic analysis of given form and returns an Expr
// that can be evaluated for result against an Env.
func (a formAnalyzer) Analyze(env core.Env, form core.Any) (core.Expr, error) {
	if IsNil(form) {
		return builtin.ConstExpr{Const: Nil{}}, nil
	}

	exp, err := macroExpand(a, env, form)
	if err != nil {
		if !errors.Is(err, builtin.ErrNoExpand) {
			return nil, err
		}

		exp = form // no expansion; use raw form
	}

	switch expr := exp.(type) {
	case Path:
		panic("PathExpr{} NOT IMPLEMENTED")
		// return PathExpr{
		// 	Root: a.root,
		// 	Path: expr,
		// }, nil

	case Symbol:
		return ResolveExpr{Symbol: expr}, nil

	case core.Seq:
		cnt, err := expr.Count()
		if err != nil {
			return nil, err
		} else if cnt == 0 {
			break
		}

		return a.analyzeSeq(env, expr)
	}

	return builtin.ConstExpr{Const: form}, nil
}

func (a formAnalyzer) analyzeSeq(env core.Env, seq core.Seq) (core.Expr, error) {
	// Analyze the call target.  This is the first item in the sequence.
	first, err := seq.First()
	if err != nil {
		return nil, err
	}

	// The call target may be a special form.  In this case, we need to get the
	// corresponding parser function, which will take care of parsing/analyzing
	// the tail.
	if sym, ok := first.(Symbol); ok {
		s, err := sym.Raw.Symbol()
		if err != nil {
			return nil, err
		}

		if parse, found := a.specialForms[s]; found {
			next, err := seq.Next()
			if err != nil {
				return nil, err
			}

			return parse(a, env, next)
		}
	}

	// Call target is not a special form and must be a Invokable.  Analyze
	// the arguments and create an InvokeExpr.
	ie := builtin.InvokeExpr{Name: fmt.Sprintf("%s", first)}
	err = core.ForEach(seq, func(item core.Any) (done bool, err error) {
		if ie.Target == nil {
			ie.Target, err = a.Analyze(env, first)
			return
		}

		var arg core.Expr
		if arg, err = a.Analyze(env, item); err == nil {
			ie.Args = append(ie.Args, arg)
		}
		return
	})
	return ie, err
}

func macroExpand(a core.Analyzer, env core.Env, form core.Any) (core.Any, error) {
	lst, ok := form.(core.Seq)
	if !ok {
		return nil, builtin.ErrNoExpand
	}

	first, err := lst.First()
	if err != nil {
		return nil, err
	}

	var target core.Any
	sym, ok := first.(Symbol)
	if ok {
		v, err := ResolveExpr{Symbol: sym}.Eval(env)
		if err != nil {
			return nil, builtin.ErrNoExpand
		}
		target = v
	}

	fn, ok := target.(builtin.Fn) // TODO(XXX):  how can builtin.Fn be capnp compatible?
	if !ok || !fn.Macro {
		return nil, builtin.ErrNoExpand
	}

	sl, err := core.ToSlice(lst)
	if err != nil {
		return nil, err
	}

	res, err := fn.Invoke(sl[1:]...)
	if err != nil {
		return nil, err
	}
	return res, nil
}
