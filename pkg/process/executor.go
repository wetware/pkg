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

type Executor api.Executor

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

// WASMExecutor is a server type for the Executor capability that
// spawns WebAssembly (WASM) processes.
type WASMExecutor struct {
	Log     log.Logger
	Runtime wazero.Runtime
}

// Executor provides the Executor capability.
func (wx *WASMExecutor) Executor() api.Executor {
	return api.Executor_ServerToClient(wx)
}

// Spawn a process by creating a process server and converting it into
// a capability as a response to the call.
func (wx *WASMExecutor) Spawn(ctx context.Context, call api.Executor_spawn) error {
	if wx.Log == nil {
		wx.Log = log.New()
	}

	if wx.Runtime == nil {
		wx.Runtime = defaultRuntime()
	}

	binary, err := call.Args().Binary()
	if err != nil {
		return err
	}
	entryFunction, err := call.Args().Entryfunction()
	if err != nil {
		return err
	}
	proc, err := wx.spawnProcess(ctx, binary, entryFunction)
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

func defaultRuntime() wazero.Runtime {
	ctx := context.Background()
	config := wazero.NewRuntimeConfig()
	r := wazero.NewRuntimeWithConfig(ctx, config)
	wasi_snapshot_preview1.MustInstantiate(ctx, r)
	return r
}

// spawnProcess creates and returns a Process that will run in e.runtime.
func (wx *WASMExecutor) spawnProcess(ctx context.Context, binary []byte, entryFunction string) (*Process, error) {
	modId := moduleId(binary) + randomId() // TODO mikel
	procIo := newIo()

	config := wazero.
		NewModuleConfig().
		WithName(modId).
		WithStdin(procIo.inR).
		WithStdout(procIo.outW).
		WithStderr(procIo.errW)

	instance := wx.Runtime.Module(modId)
	if instance == nil {
		module, err := wx.Runtime.CompileModule(ctx, binary)
		if err != nil {
			return nil, err
		}
		instance, err = wx.Runtime.InstantiateModule(ctx, module, config)
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
		logger:       wx.Log,
		releaseFuncs: make([]capnp.ReleaseFunc, 0),

		exitWaiters: make([]chan struct{}, 0),
		runDone:     make(chan error, 1),
		runContext:  runContext,
		runCancel:   runCancel,
	}

	return &proc, nil
}
