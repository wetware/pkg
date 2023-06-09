package ww

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"runtime"

	// "github.com/spy16/slurp"
	// "github.com/spy16/slurp/core"
	// "github.com/spy16/slurp/reader"
	// "github.com/spy16/slurp/repl"

	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	casm "github.com/wetware/casm/pkg"

	// "github.com/wetware/ww/api"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/system"
	"go.uber.org/fx"
)

const Version = "0.1.0"

type Ww struct {
	fx.In `ignore-unexported:"true"`

	Log    log.Logger
	Name   string
	Stdin  io.Reader `name:"stdin"`
	Stdout io.Writer `name:"stdout"`
	Stderr io.Writer `name:"stderr"`
	ROM    system.ROM
	Vat    casm.Vat
	Root   *anchor.Node
}

// String returns the cluster namespace in which the wetware is
// executing.  It is guaranteed to return ww.Name.
func (ww Ww) String() string {
	return ww.Name
}

// Exec compiles and runs the ww instance's ROM in a WASM runtime.
// It returns any error produced by the compilation or execution of
// the ROM.
func (ww Ww) Exec(ctx context.Context) error {
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

	// Instantiate Wetware.
	host, err := system.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer host.Close(ctx)

	// Compile guest module.
	//
	// TODO:  the ROM needs to be validated upstream of this call.
	compiled, err := r.CompileModule(ctx, ww.ROM)
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
		WithEnv("ns", ww.Name).
		WithStdin(ww.Stdin). // notice:  we connect stdio to host process' stdio
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr))
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	// Bind the host module to the guest module, producing a bidirectional byte-
	// stream between them.
	conn, err := host.Bind(ctx, mod)
	if err != nil {
		return err
	}
	ctx = system.WithConn(ctx, conn)

	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	// TODO(performance):  fn.CallWithStack(ctx, nil)
	_, err = fn.Call(ctx)
	return err
}
