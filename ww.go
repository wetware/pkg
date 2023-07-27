package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"io"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
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
func (ww Ww) Exec(ctx context.Context, rom ROM) error {
	// Spawn a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)

	// Instantiate WASI.
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	// TODO:  serve ww.Client over RPC connection to guest

	// Compile guest module.
	compiled, err := r.CompileModule(ctx, rom.bytecode)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	// Instantiate the guest module, and configure host exports.
	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start while instanitating.
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithEnv("ns", ww.String()).
		WithStdin(ww.Stdin). // notice:  we connect stdio to host process' stdio
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr))
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	// TODO(performance):  fn.CallWithStack(ctx, nil)
	_, err = fn.Call(ctx)
	return err
}
