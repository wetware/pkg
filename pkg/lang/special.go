package lang

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/spy16/parens"
	ww "github.com/wetware/ww/pkg"
)

var (
	_ = parens.ParseSpecial(parseDefExpr)
	_ = parens.ParseSpecial(parseQuoteExpr)
	_ = parens.ParseSpecial(parsePop)
	_ = parens.ParseSpecial(parseConj)

	_ = parens.ParseSpecial(anchorClient{}.Ls)
	_ = parens.ParseSpecial(anchorClient{}.Go)
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

type anchorClient struct{ root ww.Anchor }

func (c anchorClient) Ls(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
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

func (c anchorClient) Go(env *parens.Env, args parens.Seq) (parens.Expr, error) {
	first, err := args.First()
	if err != nil {
		return nil, err
	}

	if first == nil {
		return nil, parens.Error{
			Cause: errors.New("go expr requires at least one argument"),
		}
	}

	if args, err = args.Next(); err != nil {
		return nil, err
	}

	if p, ok := first.(Path); ok {
		return c.GoRemote(env, p, args)
	}

	return goLocal(env, first, args)
}

func (c anchorClient) GoRemote(env *parens.Env, p Path, args parens.Seq) (parens.Expr, error) {
	first, err := args.First()
	if err != nil {
		return nil, err
	}

	if _, ok := first.(Keyword); ok {
		// TODO(enhancement):  it's config. can be UNIX or Docker.
		//
		// 1. read args into map
		// 2. analyze the map & determine next steps
		return nil, errors.New("NOT IMPLEMENTED")
	}

	// TODO(enhancement):  evaluate the value in a native goroutine (on the remote host)
	return nil, errors.New("NOT IMPLEMENTED")
}

func goLocal(env *parens.Env, target parens.Any, args parens.Seq) (parens.Expr, error) {
	/*
		TODO(enhancement):  support for local UNIX procs and Docker containers.

		Read in args and check if they satisfy a exec.Cmd, or Docker equivalent.
	*/

	return parens.GoExpr{Value: target}, nil
}

func parsePop(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
	v, err := args.First()
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, parens.Error{
			Cause: errors.New("pop requires exactly one argument"),
		}
	}

	v, err = Pop(v)
	return parens.ConstExpr{Const: v}, err
}

func parseConj(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
	v, err := args.First()
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, parens.Error{
			Cause: errors.New("pop requires at least one argument"),
		}
	}

	if args, err = args.Next(); err != nil {
		return nil, err
	}

	v, err = Conj(v, args)
	return parens.ConstExpr{Const: v}, err
}
