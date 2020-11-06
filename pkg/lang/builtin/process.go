package builtin

import (
	"context"

	"github.com/wetware/ww/internal/api"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/lang/core"
	"github.com/wetware/ww/pkg/mem"
	capnp "zombiezen.com/go/capnproto2"
)

var (
	goProcType Keyword

	_ core.Process = (*RemoteProcess)(nil)
)

func init() {
	var err error
	if goProcType, err = NewKeyword(capnp.SingleSegment(nil), "go"); err != nil {
		panic(err)
	}
}

// RemoteProcess is running on a remote host.
type RemoteProcess struct{ mem.Value }

// Wait .
func (p RemoteProcess) Wait(ctx context.Context) error {
	f, done := p.Raw.Proc().Wait(ctx, nil)
	defer done()

	select {
	case <-f.Done():
	case <-ctx.Done():
		return ctx.Err()
	}

	_, err := f.Struct()
	return err
}

type procArgs []ww.Any

func (as procArgs) Remote() (p Path, ok bool) {
	if ok = as.isRemoteProc(); ok {
		p.Value = as[0].MemVal()
	}

	return
}

func (as procArgs) Args() []ww.Any {
	if as.isRemoteProc() {
		as = as.tail() // pop the path argument off the front
	}

	return as.ensureProcType() // ensure we have a process type argument (e.g. ":go")
}

func (as procArgs) isRemoteProc() bool {
	return as[0].MemVal().Type() == api.Value_Which_path
}

func (as procArgs) tail() procArgs { return as[1:] }

func (as procArgs) ensureProcType() procArgs {
	if as[0].MemVal().Type() == api.Value_Which_keyword {
		return as
	}

	return append([]ww.Any{goProcType}, as...)
}
