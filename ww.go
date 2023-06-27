package ww

import (
	"context"
	"io"

	"capnproto.org/go/capnp"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	api "github.com/wetware/ww/internal/api/process"
	"github.com/wetware/ww/pkg/csp"
)

const Version = "0.1.0"

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww struct {
	NS     string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Client capnp.Client
}

// String returns the cluster namespace in which the wetware is
// executing. If ww.NS has been assigned a non-empty string, it
// returns the string unchanged.  Else, it defaults to "ww".
func (ww *Ww) String() string {
	if ww.NS != "" {
		return ww.NS
	}

	return "ww"
}

// Exec compiles and runs the ww instance's ROM in a WASM runtime.
// It returns any error produced by the compilation or execution of
// the ROM.
func (ww Ww) Exec(ctx context.Context) error {
	runtimeCfg := wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true)
	wasmRuntime := wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
	c, err := wasi_snapshot_preview1.Instantiate(ctx, wasmRuntime)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	r := csp.Runtime{
		Runtime: wasmRuntime,
	}
	executor := api.Executor_ServerToClient(r)

	exec, release := executor.Exec(ctx, func(e api.Executor_exec_Params) error {
		return e.SetBytecode(ww.ROM)
	})
	defer release()
	<-exec.Done()

	result, err := exec.Struct()
	if err != nil {
		return err
	}
	proc := result.Process()
	wait, release := proc.Wait(ctx, nil)
	defer release()
	<-wait.Done()

	return nil
}
