package csp

import (
	"context"
	"errors"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero/sys"

	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/api/process"
)

var (
	ErrRunning    = errors.New("running")
	ErrNotStarted = errors.New("not started")
)

type Proc api.Process

func (p Proc) AddRef() Proc {
	return Proc(api.Process(p).AddRef())
}

func (p Proc) Release() {
	capnp.Client(p).Release()
}

func (p Proc) Kill(ctx context.Context) error {
	f, release := api.Process(p).Kill(ctx, nil)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (p Proc) Wait(ctx context.Context) error {
	f, release := api.Process(p).Wait(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return err
	}

	if code := res.ExitCode(); code != 0 {
		err = sys.NewExitError(code)
	}

	return err
}
