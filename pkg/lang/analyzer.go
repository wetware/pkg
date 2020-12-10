package lang

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spy16/slurp/builtin"
	score "github.com/spy16/slurp/core"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/mem"
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

	case core.Vector:
		return VectorExpr{
			eval:   a.Eval,
			Vector: f,
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

	if seq, err = seq.Next(); err != nil {
		return nil, err
	}

	// The call target may be a special form.  In this case, we need to get the
	// corresponding parser function, which will take care of parsing/analyzing
	// the tail.
	if mv := target.MemVal(); mv.Which() == api.Any_Which_symbol {
		s, err := mv.Symbol()
		if err != nil {
			return nil, err
		}

		if parse, found := a.special[s]; found {
			return parse(a, env, seq)
		}

		// symbol is not a special form; resolve.
		if target, err = a.Eval(env, target); err != nil {
			return nil, err
		}
	}

	// The call target is not a special form.  It is some kind of invokation.
	// Unpack & analyze the args.
	args, vargs, err := a.unpackArgs(env, seq)
	if err != nil {
		return nil, err
	}

	as := make([]core.Expr, len(args)+len(vargs))
	for i, arg := range append(args, vargs...) {
		if as[i], err = a.analyze(env, arg); err != nil {
			return nil, err
		}
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
		Message: fmt.Sprintf("'%s'", target.MemVal().Which()),
	}
}

func (a analyzer) unpackArgs(env core.Env, seq core.Seq) (args []ww.Any, vs []ww.Any, err error) {
	if args, err = core.ToSlice(seq); err != nil || len(args) == 0 {
		return
	}

	varg := args[len(args)-1]
	mv := varg.MemVal()

	// not vargs?
	if mv.Which() != api.Any_Which_symbol {
		return
	}

	var sym string
	if sym, err = mv.Symbol(); err != nil {
		return
	}

	if !strings.HasSuffix(sym, "...") {
		return
	}

	// It's a varg.  Symbol or collection literal form?
	switch sym {
	case "...":
		if len(args) < 2 {
			err = errors.New("invalid syntax (no vargs to unpack)")
			return
		}

		varg = args[len(args)-2]
		args = args[:len(args)-1]

	default:
		// foo...
		if varg, err = resolve(env, sym[:len(sym)-3]); err != nil {
			return
		}
	}

	// Evaluate the varg.  This will notably unquote sequences.
	if varg, err = a.Eval(env, varg); err != nil {
		return
	}

	// Coerce the varg into a sequence.
	switch v := varg.(type) {
	case core.Seq:
		// '(:foo :bar)...'
		seq = v

	case core.Seqable:
		// '[:foo :bar]...'
		if seq, err = v.Seq(); err != nil {
			return
		}

	default:
		err = errors.New("invalid syntax (vargs)")
		return

	}

	vs, err = core.ToSlice(seq)
	args = args[:len(args)-1] // pop last
	return
}

func (a analyzer) Eval(env core.Env, any ww.Any) (ww.Any, error) {
	expr, err := a.analyze(env, any)
	if err != nil {
		return nil, err
	}

	v, err := expr.Eval(env)
	if err == nil {
		any = v.(ww.Any)
	}

	return any, err
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
	if mv := first.MemVal(); mv.Which() == api.Any_Which_symbol {
		var rex ResolveExpr
		rex.Symbol.Value = mem.Value(mv)
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

func resolve(env core.Env, symbol string) (any ww.Any, err error) {
	var v interface{}
	for env != nil {
		if v, err = env.Resolve(symbol); err != nil && !errors.Is(err, core.ErrNotFound) {
			// found symbol, or there was some unexpected error
			break
		}

		// not found in the current frame. check parent.
		env = env.Parent()
	}

	if err == nil {
		any = v.(ww.Any)
	}

	return
}
