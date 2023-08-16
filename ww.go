package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"os"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
)

const (
	Version = "0.1.0"
)

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww struct {
	NS   string
	Sock system.Socket
	Opt  rpc.Options
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
func (ww *Ww) Exec(ctx context.Context, rom rom.ROM) error {
	// Spawn a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)

	// instantiate WASI
	c, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	sys, ctx, err := system.Instantiate(ctx, r, &ww.Sock)
	if err != nil {
		return err
	}
	defer sys.Close(ctx)

	// Compile guest module.
	compiled, err := r.CompileModule(ctx, rom.Bytecode)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	// Bind the socket to the guest module's config, set other
	// configuration options, then instantiate the guest module.
	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().
		// system & runtime options
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().

		// process options
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithName(rom.String()).
		WithArgs(rom.String()).     // positional args
		WithEnv("ns", ww.String())) // keyword args
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	// We are now ready to start the system RPC conn...
	conn := rpc.NewConn(&ww.Sock, &ww.Opt)
	go func() {
		defer conn.Close()
		select {
		case <-ctx.Done():
		case <-conn.Done():
		}
	}()

	// ... and run the module.
	return ww.run(ctx, mod)
}

func (ww *Ww) run(ctx context.Context, mod api.Module) error {
	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	// TODO(performance):  fn.CallWithStack(ctx, nil)
	_, err := fn.Call(ctx)
	switch err.(*sys.ExitError).ExitCode() {
	case 0:
	case sys.ExitCodeContextCanceled:
		return context.Canceled
	case sys.ExitCodeDeadlineExceeded:
		return context.DeadlineExceeded
	default:
		slog.Default().Debug(err.Error(),
			"version", Version,
			"ns", ww.String(),
			"rom", mod.Name())
	}

	return nil
}
