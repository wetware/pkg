package ww

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"runtime"
	"strings"

	// "github.com/spy16/slurp"
	// "github.com/spy16/slurp/core"
	// "github.com/spy16/slurp/reader"
	// "github.com/spy16/slurp/repl"

	"github.com/lthibault/log"
	"github.com/stealthrocket/wazergo"
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

	host, guest := net.Pipe()
	go func() {
		defer host.Close()

		// DEMO
		//
		// We send a string to the host and then read back a response and print it.
		// At the transport level, this is a bidirectional byte-stream.
		io.Copy(host, strings.NewReader("Hello, Wetware!"))
		io.Copy(os.Stdout, host)

		<-ctx.Done() // Block until the context expires
		// -- DEMO
	}()

	// Instantiate Wetware.  Wetware is implemented as a host module, like WASI.
	sysmod := wazergo.MustInstantiate(ctx, r, system.HostModule,
		system.WithPipe(guest),
		system.WithLogger(ww.Log))

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

	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	// TODO(performance):  fn.CallWithStack(ctx, nil)
	_, err = fn.Call(wazergo.WithModuleInstance(ctx, sysmod))
	return err

}
