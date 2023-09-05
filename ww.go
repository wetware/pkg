package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"io"
	"net"
	"runtime"

	"log/slog"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"

	"github.com/wetware/pkg/auth"
	csp_server "github.com/wetware/pkg/cap/csp/server"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/util/proto"
)

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww struct {
	NS     string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Root   auth.Session
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
func (ww Ww) Exec(ctx context.Context, rom rom.ROM) error {
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

	// Compile guest module.
	compiled, err := r.CompileModule(ctx, rom.Bytecode)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}
	addr := l.Addr().(*net.TCPAddr)

	// Enables the creation of non-blocking TCP connections
	// inside the WASM module. The host will pre-open the TCP
	// port and pass it to the guest through a file descriptor.
	sockCfg := sock.NewConfig().WithTCPListener("", addr.Port)
	sockCtx := sock.WithConfig(ctx, sockCfg)
	l.Close()

	// Instantiate the guest module, and configure host exports.
	mod, err := r.InstantiateModule(sockCtx, compiled, wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start while instanitating.
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(rom.String()). // TODO(soon):  use content id
		WithEnv("ns", ww.String()).
		WithName(rom.String()).
		WithStdin(ww.Stdin). // notice:  we connect stdio to host process' stdio
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr))
	if err != nil {
		return err
	}
	go csp_server.ServeModule(addr, ww.Root)
	defer mod.Close(ctx)

	return ww.run(ctx, mod)
}

func (ww Ww) run(ctx context.Context, mod api.Module) error {
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
			"version", proto.Version,
			"ns", ww.String(),
			"rom", mod.Name())
	}

	return nil
}
