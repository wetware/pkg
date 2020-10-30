package lang

var (
// _ = parens.ParseSpecial(parseDoExpr)
// _ = parens.ParseSpecial(parseIfExpr)
// _ = parens.ParseSpecial(parseDefExpr)
// _ = parens.ParseSpecial(parseQuoteExpr)
// _ = parens.ParseSpecial(parsePop)
// _ = parens.ParseSpecial(parseConj)

// _ = parens.ParseSpecial(anchorClient{}.Ls)
// _ = parens.ParseSpecial(anchorClient{}.Go)
)

// func parseDoExpr(env *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	var de parens.DoExpr
// 	err := parens.ForEach(args, func(item parens.Any) (bool, error) {
// 		expr, err := env.Analyze(item)
// 		if err != nil {
// 			return true, err
// 		}
// 		de.Exprs = append(de.Exprs, expr)
// 		return false, nil
// 	})

// 	return de, err
// }

// func parseIfExpr(env *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	count, err := args.Count()
// 	if err != nil {
// 		return nil, err
// 	} else if count != 2 && count != 3 {
// 		return nil, core.Error{
// 			Cause:   errors.New("invalid if form"),
// 			Message: fmt.Sprintf("requires 2 or 3 arguments, got %d", count),
// 		}
// 	}

// 	exprs := [3]parens.Expr{}
// 	for i := 0; i < count; i++ {
// 		f, err := args.First()
// 		if err != nil {
// 			return nil, err
// 		}

// 		expr, err := env.Analyze(f)
// 		if err != nil {
// 			return nil, err
// 		}
// 		exprs[i] = expr

// 		args, err = args.Next()
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return IfExpr{
// 		Test: exprs[0],
// 		Then: exprs[1],
// 		Else: exprs[2],
// 	}, nil
// }

// func parseQuoteExpr(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	if count, err := args.Count(); err != nil {
// 		return nil, err
// 	} else if count != 1 {
// 		return nil, core.Error{
// 			Cause:   errors.New("invalid quote form"),
// 			Message: fmt.Sprintf("requires exactly 1 argument, got %d", count),
// 		}
// 	}

// 	form, err := args.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return parens.QuoteExpr{Form: form}, nil
// }

// func parseDefExpr(env *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	if count, err := args.Count(); err != nil {
// 		return nil, err
// 	} else if count != 2 {
// 		return nil, core.Error{
// 			Cause:   errors.New("invalid def form"),
// 			Message: fmt.Sprintf("requires exactly 2 arguments, got %d", count),
// 		}
// 	}

// 	first, err := args.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	sym, ok := first.(Symbol)
// 	if !ok {
// 		return nil, core.Error{
// 			Cause:   errors.New("invalid def form"),
// 			Message: fmt.Sprintf("first arg must be symbol, not '%s'", reflect.TypeOf(first)),
// 		}
// 	}

// 	rest, err := args.Next()
// 	if err != nil {
// 		return nil, err
// 	}

// 	second, err := rest.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	res, err := env.Analyze(second)
// 	if err != nil {
// 		return nil, err
// 	}

// 	name, err := sym.Raw.Symbol()
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &parens.DefExpr{
// 		Env:   env,
// 		Name:  name,
// 		Value: res,
// 	}, nil
// }

// type anchorClient struct{ root ww.Anchor }

// func (c anchorClient) Ls(_ *parens.Env, seq parens.Seq) (parens.Expr, error) {
// 	var args []ww.Any
// 	if err := parens.ForEach(seq, func(item parens.Any) (bool, error) {
// 		args = append(args, item.(ww.Any))
// 		return false, nil
// 	}); err != nil {
// 		return nil, err
// 	}

// 	pexpr := PathExpr{Root: c.root, Path: rootPath}
// 	for _, arg := range args {
// 		if arg.MemVal().Type() == api.Value_Which_path {
// 			pexpr.Path = args[0].(Path)
// 			args = args[1:]
// 		}

// 		break
// 	}

// 	// TODO(enhancement):  other args like `:long` or `:recursive`

// 	return PathListExpr{
// 		PathExpr: pexpr,
// 		Args:     args,
// 	}, nil
// }

// func (c anchorClient) Go(env *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	n, err := args.Count()
// 	if err != nil {
// 		return nil, err
// 	}

// 	if n == 0 {
// 		return nil, errors.Errorf("expected at least one argument, got %d", n)
// 	}

// 	as := make(procArgs, 0, n)
// 	parens.ForEach(args, func(item parens.Any) (bool, error) {
// 		as = append(as, item.(ww.Any))
// 		return false, nil
// 	})

// 	if p, ok := as.Global(); ok {
// 		return GlobalGoExpr{
// 			Root: c.root,
// 			Path: p,
// 			Args: as.Args(),
// 		}, nil
// 	}

// 	return LocalGoExpr{
// 		Env:  env,
// 		Args: as.Args(),
// 	}, nil
// }

// func parsePop(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	v, err := args.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	if v == nil {
// 		return nil, core.Error{
// 			Cause: errors.New("pop requires exactly one argument"),
// 		}
// 	}

// 	v, err = Pop(v.(ww.Any))
// 	return parens.ConstExpr{Const: v}, err
// }

// func parseConj(_ *parens.Env, args parens.Seq) (parens.Expr, error) {
// 	v, err := args.First()
// 	if err != nil {
// 		return nil, err
// 	}

// 	if v == nil {
// 		return nil, core.Error{
// 			Cause: errors.New("pop requires at least one argument"),
// 		}
// 	}

// 	if args, err = args.Next(); err != nil {
// 		return nil, err
// 	}

// 	v, err = Conj(v.(ww.Any), args)
// 	return parens.ConstExpr{Const: v}, err
// }
