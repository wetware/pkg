package ww

import (
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"
	"strings"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
)

type SystemError interface {
	error
	ExitCode() uint32
}

// Ww is the execution context for WebAssembly (WASM) bytecode,
// allowing it to interact with (1) the local host and (2) the
// cluster environment.
type Ww struct {
	NS             string
	Env, Args      []string
	Stdout, Stderr io.WriteCloser
	Sess           auth.Session
	Cache          wazero.CompilationCache
	LogLevel       slog.Level
}

// String returns the cluster namespace in which the wetware is
// executing. If ww.NS has been assigned a non-empty string, it
// returns the string unchanged.  Else, it defaults to "ww".
//
// This may change in the future.
func (ww *Ww) String() string {
	if ww.NS != "" {
		return ww.NS
	}

	return "ww"
}

func (ww Ww) Logger() *slog.Logger {
	return slog.Default().With(
		"ns", ww.NS)
}

func (ww Ww) NewRuntime(ctx context.Context) wazero.Runtime {
	if ww.Cache == nil {
		ww.Cache = wazero.NewCompilationCache()
	}

	return wazero.NewRuntimeWithConfig(ctx, wazero.
		NewRuntimeConfigCompiler().
		WithCompilationCache(ww.Cache).
		WithCloseOnContextDone(true))
}

// Exec compiles and runs the ww instance's ROM in a WASM runtime.
// It returns any error produced by the compilation or execution of
// the ROM.
func (ww Ww) Exec(ctx context.Context, rom rom.ROM) error {
	// Spawn a new runtime.
	r := ww.NewRuntime(ctx)
	defer r.Close(ctx)

	c, err := wasi.Instantiate(ctx, r)
	if err != nil {
		return err
	}
	defer c.Close(ctx)

	// Instantiate wetware system socket.
	host, guest := net.Pipe()
	defer host.Close()

	// Instantiate the system host module.
	sys, err := system.Instantiate(ctx, r, guest)
	if err != nil {
		return err
	}
	defer sys.Close(ctx)
	ctx = wazergo.WithModuleInstance(ctx, sys)

	// Compile guest module.
	compiled, err := r.CompileModule(ctx, rom.Bytecode)
	if err != nil {
		return err
	}
	defer compiled.Close(ctx)

	// Bind the Wetware environment to the wazero.ModuleConfig.
	mc, err := ww.BindConfig(rom, guest)
	if err != nil {
		return err
	}
	defer ww.BindSocket(host).Close()

	// Instantiate the guest module.
	mod, err := r.InstantiateModule(ctx, compiled, mc)
	if err != nil {
		return err
	}
	defer mod.Close(ctx)

	return ww.run(ctx, mod)
}

func (ww Ww) BindSocket(sock io.ReadWriteCloser) *rpc.Conn {
	server := core.Terminal_NewServer(ww.Sess)
	logger := ww.Logger().With(
		"host", ww.Sess.Peer(),
		"peer", ww.Sess.Peer(),
		"vat", ww.Sess.Vat())

	return rpc.NewConn(rpc.NewStreamTransport(sock), &rpc.Options{
		BootstrapClient: capnp.NewClient(server),
		ErrorReporter:   system.ErrorReporter{Logger: logger},
	})
}

func (ww Ww) BindConfig(rom rom.ROM, r io.Reader) (wazero.ModuleConfig, error) {
	args := append([]string{rom.String()}, ww.Args...)

	return ww.BindEnv(wazero.NewModuleConfig().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithStartFunctions(). // don't automatically call _start while instanitating.
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(args...).
		WithName(rom.String()).
		WithStdout(ww.Stdout).
		WithStderr(ww.Stderr).
		WithStdin(r))
}

func (ww Ww) BindEnv(mc wazero.ModuleConfig) (wazero.ModuleConfig, error) {
	return bindEnv(mc.
		WithEnv("ns", ww.String()).
		WithEnv("WW_LOGLVL", ww.LogLevel.String()),
		ww.Env)
}

func bindEnv(mc wazero.ModuleConfig, vars []string) (wazero.ModuleConfig, error) {
	for _, s := range vars {
		if ss, ok := parseEnvVar(s); ok {
			mc = mc.WithEnv(ss[0], ss[1])
		} else {
			return nil, fmt.Errorf("invalid env var: %s", ss)
		}
	}

	return mc, nil
}

func parseEnvVar(s string) (ss [2]string, ok bool) {
	if kv := strings.SplitN(s, "=", 1); len(kv) == 2 {
		ss[0], ss[1] = kv[0], kv[1]
		ok = true
	}

	return
}

func (ww Ww) run(ctx context.Context, mod api.Module) error {
	// Grab the the main() function and call it with the system context.
	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return errors.New("missing export: _start")
	}

	return fn.CallWithStack(ctx, nil)
}
