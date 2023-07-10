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

// process is the main implementation of the Process capability.
type process struct {
	done   <-chan execResult
	cancel context.CancelFunc
	result execResult
}

func (p process) Kill(context.Context, api.Process_kill) error {
	p.cancel()
	return nil
}

func (p *process) Wait(ctx context.Context, call api.Process_wait) error {
	select {
	case res, ok := <-p.done:
		if ok {
			p.result = res
		}

	case <-ctx.Done():
		return ctx.Err()
	}

	res, err := call.AllocResults()
	if err == nil {
		err = p.result.Bind(res)
	}

	return err
}

type execResult struct {
	Values []uint64
	Err    error
}

func (r execResult) Bind(res api.Process_wait_Results) error {
	if r.Err != nil {
		res.SetExitCode(r.Err.(*sys.ExitError).ExitCode())
	}

	return nil
}
