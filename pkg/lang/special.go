package lang

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
)

var (
	_ = parens.ParseSpecial(parseGoExpr)
	_ = parens.ParseSpecial(parseDefExpr)
	_ = parens.ParseSpecial(parseQuoteExpr)

	_ = parens.ParseSpecial(anchorClient{}.parseLsExpr)
)

func parseQuoteExpr(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
	if count, err := args.Count(); err != nil {
		return nil, err
	} else if count != 1 {
		return nil, parens.Error{
			Cause:   errors.New("invalid quote form"),
			Message: fmt.Sprintf("requires exactly 1 argument, got %d", count),
		}
	}

	form, err := args.First()
	if err != nil {
		return nil, err
	}

	return parens.QuoteExpr{Form: form}, nil
}

func parseDefExpr(env *parens.Env, args parens.Seq) (parens.Expr, error) {
	if count, err := args.Count(); err != nil {
		return nil, err
	} else if count != 2 {
		return nil, parens.Error{
			Cause:   errors.New("invalid def form"),
			Message: fmt.Sprintf("requires exactly 2 arguments, got %d", count),
		}
	}

	first, err := args.First()
	if err != nil {
		return nil, err
	}

	sym, ok := first.(Symbol)
	if !ok {
		return nil, parens.Error{
			Cause:   errors.New("invalid def form"),
			Message: fmt.Sprintf("first arg must be symbol, not '%s'", reflect.TypeOf(first)),
		}
	}

	rest, err := args.Next()
	if err != nil {
		return nil, err
	}

	second, err := rest.First()
	if err != nil {
		return nil, err
	}

	res, err := env.Eval(second)
	if err != nil {
		return nil, err
	}

	s, err := sym.v.Symbol()
	if err != nil {
		return nil, err
	}

	return parens.DefExpr{
		Name:  s,
		Value: res,
	}, nil
}

func parseGoExpr(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
	return nil, errors.New("NOT IMPLEMENTED")
	// v, err := args.First()
	// if err != nil {
	// 	return nil, err
	// }

	// if v == nil {
	// 	return nil, parens.Error{
	// 		Cause: errors.New("go expr requires exactly one argument"),
	// 	}
	// }

	// return GoExpr{Value: v}, nil
}

type anchorClient struct{ root ww.Anchor }

func (c anchorClient) parseLsExpr(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
	v, err := args.First()
	if err != nil {
		return nil, err
	} else if v == nil {
		return nil, parens.Error{
			Cause: errors.New("ls expr requires a path argument"),
		}
	}

	p, ok := v.(Path)
	if !ok {
		return nil, parens.Error{
			Cause:   errors.New("arg 0 must be path"),
			Message: fmt.Sprintf("got %s", reflect.TypeOf(v)),
		}
	}

	return PathListExpr{PathExpr: PathExpr{
		Root: c.root,
		Path: p,
	}}, nil
}
