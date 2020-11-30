package lang

import (
	"errors"
	"fmt"
	"sync"

	"github.com/spy16/slurp/builtin"
	score "github.com/spy16/slurp/core"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
)

var _ core.Analyzer = (*analyzer)(nil)

// SpecialParser defines a special form.
type SpecialParser func(core.Analyzer, core.Env, core.Seq) (core.Expr, error)

type analyzer struct {
	root    ww.Anchor
	special map[string]SpecialParser
}

func newAnalyzer(root ww.Anchor) core.Analyzer {
	if root == nil {
		panic("nil root")
	}

	return analyzer{
		root: root,
		special: map[string]SpecialParser{
			"do": parseDo,
			"if": parseIf,
			// "fn":    parseFn,
			"def": parseDef,
			// "macro": parseMacro,
			"quote": parseQuote,
			// "go": c.Go,
			"ls": lsParser(root),
			// "pop":   parsePop,
			// "conj":  parseConj,
		},
	}
}

// Analyze performs syntactic analysis of given form and returns an Expr
// that can be evaluated for result against an Env.
func (a analyzer) Analyze(env core.Env, rawForm score.Any) (core.Expr, error) {
	form := rawForm.(ww.Any)

	if core.IsNil(form) {
		return builtin.ConstExpr{Const: core.Nil{}}, nil
	}

	exp, err := macroExpand(a, env, form)
	if err != nil {
		if !errors.Is(err, builtin.ErrNoExpand) {
			return nil, err
		}

		exp = form // no expansion; use raw form
	}

	switch expr := exp.(type) {
	case core.Path:
		return PathExpr{
			Root: a.root,
			Path: expr,
		}, nil

	case core.Symbol:
		return ResolveExpr{expr}, nil

	case core.Seq:
		cnt, err := expr.Count()
		if err != nil {
			return nil, err
		} else if cnt == 0 {
			break
		}

		return a.analyzeSeq(env, expr)
	}

	return ConstExpr{form}, nil
}

func (a analyzer) analyzeSeq(env core.Env, seq core.Seq) (core.Expr, error) {
	// Analyze the call target.  This is the first item in the sequence.
	first, err := seq.First()
	if err != nil {
		return nil, err
	}

	// The call target may be a special form.  In this case, we need to get the
	// corresponding parser function, which will take care of parsing/analyzing
	// the tail.
	if mv := first.MemVal(); mv.Type() == api.Value_Which_symbol {
		s, err := mv.Raw.Symbol()
		if err != nil {
			return nil, err
		}

		if parse, found := a.special[s]; found {
			next, err := seq.Next()
			if err != nil {
				return nil, err
			}

			return parse(a, env, next)
		}
	}

	// Call target is not a special form and must be a Invokable. Analyze
	// the arguments and create an InvokeExpr.
	ie := InvokeExpr{
		Name: fmt.Sprintf("%v", first),
	}
	err = core.ForEach(seq, func(item ww.Any) (done bool, err error) {
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

func macroExpand(a core.Analyzer, env core.Env, form ww.Any) (ww.Any, error) {
	// TODO:  function calls
	return nil, builtin.ErrNoExpand

	// lst, ok := form.(core.Seq)
	// if !ok {
	// 	return nil, builtin.ErrNoExpand
	// }

	// first, err := lst.First()
	// if err != nil {
	// 	return nil, err
	// }

	// var target ww.Any
	// sym, ok := first.(Symbol)
	// if ok {
	// 	v, err := ResolveExpr{Symbol: sym}.Eval(env)
	// 	if err != nil {
	// 		return nil, builtin.ErrNoExpand
	// 	}
	// 	target = v.(ww.Any)
	// }

	// fn, ok := target.(Fn) // TODO(XXX):  how can builtin.Fn be capnp compatible?
	// if !ok || !fn.Macro {
	// 	return nil, builtin.ErrNoExpand
	// }

	// sl, err := core.ToSlice(lst)
	// if err != nil {
	// 	return nil, err
	// }

	// res, err := fn.Invoke(sl[1:]...)
	// if err != nil {
	// 	return nil, err
	// }
	// return res, nil
}

type linker struct {
	mu sync.RWMutex
	vs map[string]ww.Any
}

func (l *linker) Resolve(link string) (ww.Any, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if v, ok := l.vs[link]; ok {
		return v, nil
	}

	return nil, core.Error{
		Message: link,
		Cause:   core.ErrNotFound,
	}
}
