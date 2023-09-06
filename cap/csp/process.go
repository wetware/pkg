package csp

import (
	"context"
	"errors"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero/sys"

	api "github.com/wetware/pkg/api/process"
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

// Kill a process and any sub processes it might have spawned.
func (p Proc) Kill(ctx context.Context) error {
	f, release := api.Process(p).Kill(ctx, nil)
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := f.Struct()
	if err != nil {
		return err
	}
	return nil
}

func (p Proc) Wait(ctx context.Context) error {
	f, release := api.Process(p).Wait(ctx, nil)
	defer release()

	select {
	case <-f.Done():
	case <-ctx.Done():
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	res, err := f.Struct()
	if err != nil {
		return err
	}

	if code := res.ExitCode(); code != 0 {
		err = sys.NewExitError(code)
	}

	return err
}
