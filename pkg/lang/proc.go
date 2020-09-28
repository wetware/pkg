package lang

import (
	"github.com/pkg/errors"
	"github.com/spy16/parens"
	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	capnp "zombiezen.com/go/capnproto2"
)

var goProcType Keyword

func init() {
	var err error
	if goProcType, err = NewKeyword(capnp.SingleSegment(nil), "go"); err != nil {
		panic(err)
	}
}

type procArgs []ww.Any

func newProcArgs(args parens.Seq) (procArgs, error) {
	n, err := args.Count()
	if err != nil {
		return nil, err
	}

	if n == 0 {
		return nil, errors.Errorf("expected at least one argument, got %d", n)
	}

	as := make([]ww.Any, 0, n)
	parens.ForEach(args, func(item parens.Any) (bool, error) {
		as = append(as, item.(ww.Any))
		return false, nil
	})

	return as, nil
}

func (as procArgs) Global() (p Path, ok bool) {
	if ok = as.isGlobalProc(); ok {
		p.Value = as[0].MemVal()
	}

	return
}

func (as procArgs) Args() []ww.Any {
	if as.isGlobalProc() {
		as = as.tail() // pop the path argument off the front
	}

	return as.ensureProcType() // ensure we have a process type argument (e.g. ":go")
}

func (as procArgs) isGlobalProc() bool {
	return as[0].MemVal().Type() == api.Value_Which_path
}

func (as procArgs) tail() procArgs { return as[1:] }

func (as procArgs) ensureProcType() procArgs {
	if as[0].MemVal().Type() == api.Value_Which_keyword {
		return as
	}

	return append([]ww.Any{goProcType}, as...)
}
