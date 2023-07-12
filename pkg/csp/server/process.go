package server

import (
	"context"
	"errors"

	"github.com/tetratelabs/wazero/sys"

	api "github.com/wetware/ww/api/process"
)

var (
	ErrRunning    = errors.New("running")
	ErrNotStarted = errors.New("not started")
)

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
