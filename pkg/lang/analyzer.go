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

func newAnalyzer(root ww.Anchor, paths []string) (core.Analyzer, error) {
	if root == nil {
		return nil, errors.New("nil anchor")
	}

	return analyzer{
		root: root,
		special: map[string]SpecialParser{
			"do":    parseDo,
			"if":    parseIf,
			"def":   parseDef,
			"fn":    parseFn,
			"macro": parseMacro,
			"quote": parseQuote,
			// "go": c.Go,
			"ls":     lsParser(root),
			"eval":   parseEval,
			"import": importer(paths).Parse,
		},
	}, nil
}

// Analyze performs syntactic analysis of given form and returns an Expr
// that can be evaluated for result against an Env.
func (a analyzer) Analyze(env core.Env, rawForm score.Any) (core.Expr, error) {
	return a.analyze(env, rawForm.(ww.Any))
}

// analyze allows private methods of `analyzer` to by pass the initial
// type assertion for `ww.Any`.
func (a analyzer) analyze(env core.Env, any ww.Any) (core.Expr, error) {
	if core.IsNil(any) {
		return builtin.ConstExpr{Const: core.Nil{}}, nil
	}

	form, err := a.macroExpand(env, any)
	if err != nil {
		if !errors.Is(err, builtin.ErrNoExpand) {
			return nil, err
		}

		form = any // no expansion; use unmodified form
	}

	switch f := form.(type) {
	case core.Symbol:
		return ResolveExpr{f}, nil

	case core.Path:
		return PathExpr{
			Root: a.root,
			Path: f,
		}, nil

	case core.Seq:
		return a.analyzeSeq(env, f)

	}

	return ConstExpr{form}, nil
}

func (a analyzer) analyzeSeq(env core.Env, seq core.Seq) (core.Expr, error) {
	// Return an empty sequence unmodified.
	cnt, err := seq.Count()
	if err != nil || cnt == 0 {
		return ConstExpr{seq}, err
	}

	// Analyze the call target.  This is the first item in the sequence.
	// Call targets come in several flavors.
	target, err := seq.First()
	if err != nil {
		return nil, err
	}

	args, err := seq.Next()
	if err != nil {
		return nil, err
	}

	// The call target may be a special form.  In this case, we need to get the
	// corresponding parser function, which will take care of parsing/analyzing
	// the tail.
	if mv := target.MemVal(); mv.Type() == api.Value_Which_symbol {
		s, err := mv.Raw.Symbol()
		if err != nil {
			return nil, err
		}

		if parse, found := a.special[s]; found {
			return parse(a, env, args)
		}

		// symbol is not a special form; resolve.
		expr, err := a.analyze(env, target)
		if err != nil {
			return nil, err
		}

		v, err := expr.Eval(env)
		if err != nil {
			return nil, err
		}

		target = v.(ww.Any)
	}

	// The call target is not a special form.  It is some kind of invokation.
	// Start by analyzing its arguments.
	as := make([]core.Expr, 0, cnt-1)
	if err = core.ForEach(args, func(item ww.Any) (bool, error) {
		arg, err := a.analyze(env, item)
		as = append(as, arg)
		return false, err
	}); err != nil {
		return nil, err
	}

	// Determine whether this is an invokation on a Fn or an invokable
	// value, and return the appropriate expression.
	switch t := target.(type) {
	case core.Fn:
		return CallExpr{
			Fn:       t,
			Analyzer: a,
			Args:     as,
		}, nil

	case core.Invokable:
		return InvokeExpr{
			Target: t,
			Args:   as,
		}, nil

	}

	return nil, core.Error{
		Cause:   core.ErrNotInvokable,
		Message: fmt.Sprintf("'%s'", target.MemVal().Type()),
	}
}

func (a analyzer) macroExpand(env core.Env, form ww.Any) (ww.Any, error) {
	seq, ok := form.(core.Seq)
	if !ok {
		return nil, builtin.ErrNoExpand
	}

	cnt, err := seq.Count()
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return nil, builtin.ErrNoExpand
	}

	first, err := seq.First()
	if err != nil {
		return nil, err
	}

	var v interface{}
	if mv := first.MemVal(); mv.Type() == api.Value_Which_symbol {
		var rex ResolveExpr
		rex.Symbol.Raw = mv.Raw
		if v, err = rex.Eval(env); err != nil {
			return nil, builtin.ErrNoExpand
		}
	}

	fn, ok := v.(core.Fn)
	if !ok || !fn.Macro() {
		return nil, builtin.ErrNoExpand
	}

	args, err := seq.Next()
	if err != nil {
		return nil, err
	}

	as := make([]core.Expr, 0, cnt-1)
	if err = core.ForEach(args, func(item ww.Any) (bool, error) {
		arg, err := a.analyze(env, item)
		as = append(as, arg)
		return false, err
	}); err != nil {
		return nil, err
	}

	if v, err = (CallExpr{
		Fn:       fn,
		Analyzer: a,
		Args:     as,
	}).Eval(env); err != nil {
		return nil, err
	}

	return v.(ww.Any), nil
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
