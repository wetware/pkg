package csp_server

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"

	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/csp/proc"
)

// Runtime is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Runtime panics.
type Runtime struct {
	Runtime    wazero.Runtime
	HostModule *wazergo.ModuleInstance[*proc.Module]
}

// Executor provides the Executor capability.
func (r Runtime) Executor() csp.Executor {
	return csp.Executor(api.Executor_ServerToClient(r))
}

func (r Runtime) Exec(ctx context.Context, call api.Executor_exec) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	p, err := r.mkproc(ctx, call.Args())
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Runtime) mkproc(ctx context.Context, args api.Executor_exec_Params) (*process, error) {
	mod, err := r.mkmod(ctx, args)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	done, cancel := r.spawn(fn)
	return &process{
		done:   done,
		cancel: cancel,
	}, nil
}

func (r Runtime) mkmod(ctx context.Context, args api.Executor_exec_Params) (wasm.Module, error) {
	bc, err := args.Bytecode()
	if err != nil {
		return nil, err
	}

	name := csp.ByteCode(bc).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	module, err := r.Runtime.CompileModule(ctx, bc)
	if err != nil {
		return nil, err
	}

	return r.Runtime.InstantiateModule(ctx, module, wazero.
		NewModuleConfig().
		WithName(name).
		WithStartFunctions(). // disable automatic calling of _start (main)
		WithRandSource(rand.Reader))
}

func (r Runtime) spawn(fn wasm.Function) (<-chan execResult, context.CancelFunc) {
	out := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(out)
		defer cancel()

		vs, err := fn.Call(wazergo.WithModuleInstance(ctx, r.HostModule))
		out <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return out, cancel
}
