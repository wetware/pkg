package process

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	capnp "capnproto.org/go/capnp/v3"
	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/sys"

	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/process"
)

type Proc api.Process

func (p Proc) AddRef() Proc {
	return Proc(api.Process(p).AddRef())
}

func (p Proc) Release() {
	capnp.Client(p).Release()

}

func (p Proc) Start(ctx context.Context) error {
	f, release := api.Process(p).Start(ctx, nil)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (p Proc) Stop(ctx context.Context) error {
	f, release := api.Process(p).Stop(ctx, nil)
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

	e, err := res.Error()
	if err != nil {
		return err
	}

	switch e.Which() {
	case api.Error_Which_none:
		return nil

	case api.Error_Which_exitErr:
		mod, err := e.ExitErr().Module()
		if err != nil {
			return err
		}
		return sys.NewExitError(mod, e.ExitErr().Code())
	}

	return fmt.Errorf("unknown error type: %d", e.Which())
}

// process is the main implementation of the Process capability.
type process struct {
	fn     wasm.Function
	handle procHandle
}

// Stop calls the runtime cancellation function.
func (p *process) Stop(context.Context, api.Process_stop) error {
	state := p.handle.Load()
	if state.Err == nil {
		state.Cancel()
	}

	return state.Err
}

// Start the process in the background.
func (p *process) Start(_ context.Context, call api.Process_start) error {
	state := p.handle.Load()
	if state.Err != ErrNotStarted {
		return state.Err
	}

	p.handle.Exec(p.fn)
	return nil
}

// Wait for the process to finish running.
func (p *process) Wait(ctx context.Context, call api.Process_wait) error {
	state := p.handle.Load()
	if state.Err == ErrNotStarted {
		return state.Err
	}

	results, err := call.AllocResults()
	if err != nil {
		return err
	}

	call.Go()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-state.Ctx.Done():
		return p.handle.Bind(results)
	}
}

// procHandle encapsulates all the runtime state of a process.  Its
// methods are safe for concurrent access.
type procHandle atomic.Pointer[state]

var (
	ErrRunning    = errors.New("running")
	ErrNotStarted = errors.New("not started")
)

// Exec sets the current state to ErrRunning, calls the function, and
// then sets the current state to the resulting error.
func (as *procHandle) Exec(fn wasm.Function) {
	ctx, cancel := context.WithCancel(context.Background())

	// set "running" state
	(*atomic.Pointer[state])(as).Store(&state{
		Ctx:    ctx,
		Cancel: cancel,
		Err:    ErrRunning,
	})

	go func() {
		defer cancel()

		// block until function call completes
		_, err := fn.Call(ctx)

		// call entrypoint function & set "finished" state
		(*atomic.Pointer[state])(as).Store(&state{
			Ctx:    ctx,
			Cancel: cancel,
			Err:    err,
		})
	}()
}

// Bind the error from the entrypoint function to the results struct.
// Callers MUST NOT call Bind until the function has returned.
func (as *procHandle) Bind(res api.Process_wait_Results) error {
	state := as.Load()
	if state.Err == nil {
		return nil
	}

	e, err := res.NewError()
	if err != nil {
		return err
	}

	ee := state.Err.(*sys.ExitError)
	e.SetExitErr()
	e.ExitErr().SetCode(ee.ExitCode())
	return e.ExitErr().SetModule(ee.ModuleName())
}

// Load the current state atomically.  The resulting resulting state
// defaults to ErrNotStarted.
func (as *procHandle) Load() state {
	if s := (*atomic.Pointer[state])(as).Load(); s != nil {
		return *s
	}

	return state{
		Cancel: func() {},
		Err:    ErrNotStarted,
	}
}

type state struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Err    error
}
