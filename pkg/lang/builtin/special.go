package builtin

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spy16/slurp"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	_ = SpecialParser(parseDo)
	_ = SpecialParser(parseIf)
	_ = SpecialParser(parseQuote)
	_ = SpecialParser(parseDef)

	// _ = SpecialParser(parseFn)
	// _ = SpecialParser(parseMacro)

	_ = SpecialParser(parseLs)
	_ = SpecialParser(parseGo)

	doSymbol Symbol
)

func init() {
	var err error
	if doSymbol, err = NewSymbol(capnp.SingleSegment(nil), "do"); err != nil {
		panic(err)
	}
}

func parseDo(a *Analyzer, env core.Env, args core.Seq) (core.Expr, error) {
	var de DoExpr
	err := core.ForEach(args, func(item ww.Any) (bool, error) {
		expr, err := a.Analyze(env, item)
		if err != nil {
			return true, err
		}
		de.Exprs = append(de.Exprs, expr)
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return de, nil
}

func parseIf(a *Analyzer, env core.Env, args core.Seq) (core.Expr, error) {
	count, err := args.Count()
	if err != nil {
		return nil, err
	} else if count != 2 && count != 3 {
		return nil, core.Error{
			Cause:   fmt.Errorf("%w: if", slurp.ErrParseSpecial),
			Message: fmt.Sprintf("requires 2 or 3 arguments, got %d", count),
		}
	}

	exprs := [3]core.Expr{}
	for i := 0; i < count; i++ {
		f, err := args.First()
		if err != nil {
			return nil, err
		}

		expr, err := a.Analyze(env, f)
		if err != nil {
			return nil, err
		}
		exprs[i] = expr

		args, err = args.Next()
		if err != nil {
			return nil, err
		}
	}

	return IfExpr{
		Test: exprs[0],
		Then: exprs[1],
		Else: exprs[2],
	}, nil
}
func parseQuote(a *Analyzer, _ core.Env, args core.Seq) (core.Expr, error) {
	if count, err := args.Count(); err != nil {
		return nil, err
	} else if count != 1 {
		return nil, core.Error{
			Cause:   fmt.Errorf("%w: quote", slurp.ErrParseSpecial),
			Message: fmt.Sprintf("requires exactly 1 argument, got %d", count),
		}
	}

	first, err := args.First()
	if err != nil {
		return nil, err
	}

	return QuoteExpr{Form: first}, nil
}

func parseDef(a *Analyzer, env core.Env, args core.Seq) (core.Expr, error) {
	e := core.Error{Cause: fmt.Errorf("%w: def", slurp.ErrParseSpecial)}

	if args == nil {
		return nil, e.With("requires exactly 2 args, got 0")
	}

	if count, err := args.Count(); err != nil {
		return nil, err
	} else if count != 2 {
		return nil, e.With(fmt.Sprintf(
			"requires exactly 2 arguments, got %d", count))
	}

	first, err := args.First()
	if err != nil {
		return nil, err
	}

	sym, ok := first.(core.Symbol)
	if !ok {
		return nil, e.With(fmt.Sprintf(
			"first arg must be symbol, not '%s'", reflect.TypeOf(first)))
	}

	symStr, err := sym.Symbol()
	if err != nil {
		return nil, err
	}

	rest, err := args.Next()
	if err != nil {
		return nil, err
	}

	second, err := rest.First()
	if err != nil {
		return nil, err
	}

	res, err := a.Analyze(env, second)
	if err != nil {
		return nil, err
	}

	return DefExpr{
		Name:  symStr,
		Value: res,
	}, nil
}

func parseLs(a *Analyzer, _ core.Env, seq core.Seq) (core.Expr, error) {
	args, err := core.ToSlice(seq)
	if err != nil {
		return nil, err
	}

	pexpr := PathExpr{Root: a.root, Path: rootPath}
	for _, arg := range args {
		if arg.MemVal().Type() == api.Value_Which_path {
			pexpr.Path = args[0].(Path)
			args = args[1:]
		}

		break
	}

	// TODO(enhancement):  other args like `:long` or `:recursive`

	return PathListExpr{
		PathExpr: pexpr,
		Args:     args,
	}, nil
}

func parseGo(a *Analyzer, env core.Env, seq core.Seq) (core.Expr, error) {
	args, err := core.ToSlice(seq)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, errors.Errorf("expected at least one argument, got %d", len(args))
	}

	if p, ok := procArgs(args).Remote(); ok {
		return RemoteGoExpr{
			Root: a.root,
			Path: p,
			Args: procArgs(args).Args(),
		}, nil
	}

	return LocalGoExpr{
		Args: procArgs(args).Args(),
	}, nil
}
