package process

import (
	"context"
	"fmt"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	api "github.com/wetware/ww/internal/api/process"
)

const start = "_start" // start function called in every WASM module

// Executor contains a WASM runtime and can spawn processes in it.
type Executor struct {
	logger  log.Logger
	runtime wazero.Runtime
}

// Executor provides the Executor capability.
func (e Executor) Executor() api.Executor {
	return api.Executor_ServerToClient(e)
}

// NewExecutor is the default constructor for Executor.
func NewExecutor(ctx context.Context, logger log.Logger) Executor {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	return Executor{logger: logger, runtime: r}
}

// Close the executor runtime. Spawned processes should be inidividually closed
// calling Process.Close().
func (e Executor) Close(ctx context.Context) error {
	return e.runtime.Close(ctx)
}

// Spawn a process by creating a process server and converting it into
// a capability as a response to the call.
func (e Executor) Spawn(ctx context.Context, call api.Executor_spawn) error {
	binary, err := call.Args().Binary()
	if err != nil {
		return err
	}
	entryFunction, err := call.Args().Entryfunction()
	if err != nil {
		return err
	}
	proc, err := e.spawnProcess(ctx, binary, entryFunction)
	if err != nil {
		return err
	}
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	err = res.SetProcess(api.Process_ServerToClient(proc))

	return err
}

// spawnProcess creates and returns a Process that will run in e.runtime.
func (e Executor) spawnProcess(ctx context.Context, binary []byte, entryFunction string) (*Process, error) {
	modId := moduleId(binary) + randomId() // TODO mikel
	procIo := newIo()

	config := wazero.
		NewModuleConfig().
		WithName(modId).
		WithStdin(procIo.inR).
		WithStdout(procIo.outW).
		WithStderr(procIo.errBuffer)

	instance := e.runtime.Module(modId)
	if instance == nil {
		module, err := e.runtime.CompileModule(ctx, binary)
		if err != nil {
			return nil, err
		}
		instance, err = e.runtime.InstantiateModule(ctx, module, config)
		if err != nil {
			return nil, err
		}
		// instance.ExportedFunction(start).Call(ctx)
	}

	function := instance.ExportedFunction(entryFunction)
	if function == nil {
		return nil, fmt.Errorf("function %s not found in module %s", entryFunction, modId)
	}

	runContext, runCancel := context.WithCancel(context.TODO())

	proc := Process{
		function:     function,
		id:           processId(modId, entryFunction),
		io:           procIo,
		logger:       e.logger,
		releaseFuncs: make([]capnp.ReleaseFunc, 0),

		exitWaiters: make([]chan struct{}, 0),
		runDone:     make(chan error, 1),
		runContext:  runContext,
		runCancel:   runCancel,
	}

	return &proc, nil
}
