package lang

import (
	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/mem"
)

// ProcSpecFromArgs .
func ProcSpecFromArgs(args parens.Seq) ww.ProcSpec {
	return func(p api.Anchor_go_Params) error {
		first, err := args.First()
		if err != nil {
			return err
		}

		switch v := first.(ww.Any).Data(); v.Type() {
		case api.Value_Which_list:
			return setGoroutineSpec(p, v)

		case api.Value_Which_keyword:
			return setSpecFromKeywordArgs(p, args)

		default:
			return parens.Error{
				Cause:   errors.New("unsuitable type for remote goroutine"),
				Message: v.Type().String(),
			}

		}
	}
}

func setGoroutineSpec(p api.Anchor_go_Params, v mem.Value) error {
	spec, err := p.NewSpec()
	if err != nil {
		return err
	}

	g, err := spec.NewGoroutine()
	if err != nil {
		return err
	}

	return g.SetValue(v.Raw)
}

func setSpecFromKeywordArgs(p api.Anchor_go_Params, args parens.Seq) error {
	return errors.New("NOT IMPLEMENTED")
}

func goLocal(env *parens.Env, target parens.Any, args parens.Seq) (parens.Expr, error) {
	/*
		TODO(enhancement):  support for local UNIX procs and Docker containers.

		Read in args and check if they satisfy a exec.Cmd, or Docker equivalent.
	*/

	return parens.GoExpr{Value: target}, nil
}

func (c anchorClient) GoRemote(env *parens.Env, p Path, args parens.Seq) (parens.Expr, error) {
	return RemoteProcExpr{
		PathExpr: PathExpr{Root: c.root, Path: p},
		Spec:     ProcSpecFromArgs(args),
	}, nil
}
