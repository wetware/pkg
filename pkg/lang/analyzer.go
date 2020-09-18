package lang

import (
	"fmt"

	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
)

var (
	_ parens.Analyzer = formAnalyzer{}
)

type formAnalyzer struct {
	root         ww.Anchor
	specialForms map[string]parens.ParseSpecial
}

func analyzer(root ww.Anchor) formAnalyzer {
	c := anchorClient{root}

	return formAnalyzer{
		root: root,
		specialForms: map[string]parens.ParseSpecial{
			"def":   parseDefExpr,
			"quote": parseQuoteExpr,
			"go":    c.Go,
			"ls":    c.Ls,
			"pop":   parsePop,
			"conj":  parseConj,
		},
	}
}

// Analyze performs syntactic analysis of given form and returns an Expr
// that can be evaluated for result against an Env.
func (a formAnalyzer) Analyze(env *parens.Env, form parens.Any) (parens.Expr, error) {
	if parens.IsNil(form) {
		return &parens.ConstExpr{Const: Nil{}}, nil
	}

	switch f := form.(type) {
	case Path:
		return PathExpr{
			Root: a.root,
			Path: f,
		}, nil

	case Symbol:
		sym, err := f.v.Symbol()
		if err != nil {
			return nil, err
		}

		v := env.Resolve(sym)
		if v == nil {
			// If the symbol maps to a special form, continue.
			// It will be treated as a ConstExpr.
			if _, ok := a.specialForms[sym]; !ok {
				return nil, parens.Error{
					Cause:   parens.ErrNotFound,
					Message: sym,
				}
			}
		}

	case parens.Seq:
		cnt, err := f.Count()
		if err != nil {
			return nil, err
		} else if cnt == 0 {
			break
		}

		return a.analyzeSeq(env, f)
	}

	return parens.ConstExpr{Const: form}, nil
}

func (a formAnalyzer) analyzeSeq(env *parens.Env, seq parens.Seq) (parens.Expr, error) {
	//	Analyze the call target.  This is the first item in the sequence.
	first, err := seq.First()
	if err != nil {
		return nil, err
	}

	// The call target may be a special form.  In this case, we need to get the
	// corresponding parser function, which will take care of parsing/analyzing
	// the tail.
	if sym, ok := first.(Symbol); ok {
		s, err := sym.v.Symbol()
		if err != nil {
			return nil, err
		}

		if parse, found := a.specialForms[s]; found {
			next, err := seq.Next()
			if err != nil {
				return nil, err
			}

			return parse(env, next)
		}
	}

	// Call target is not a special form and must be a Invokable.  Analyze
	// the arguments and create an InvokeExpr.
	ie := parens.InvokeExpr{Name: fmt.Sprintf("%s", first)}
	err = parens.ForEach(seq, func(item parens.Any) (done bool, err error) {
		if ie.Target == nil {
			ie.Target, err = a.Analyze(env, first)
			return
		}

		var arg parens.Expr
		if arg, err = a.Analyze(env, item); err == nil {
			ie.Args = append(ie.Args, arg)
		}
		return
	})
	return ie, err
}
