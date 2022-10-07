package wasm

import (
	"context"
	"fmt"
	"io"
	"os"

	"capnproto.org/go/capnp/v3"

	"github.com/tetratelabs/wazero"
	gojs "github.com/tetratelabs/wazero/imports/go"
	"github.com/tetratelabs/wazero/sys"

	api "github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/internal/api/wasm"
)

type ConfigFunc func(api.Executor_exec_Params) error

type RuntimeFactory struct {
	Config wazero.RuntimeConfig
}

func (f RuntimeFactory) Runtime(ctx context.Context) Runtime {
	// // The Wasm binary (stars/main.wasm) is very large (>7.5MB). Use wazero's
	// // compilation cache to reduce performance penalty of multiple runs.
	// compilationCacheDir := ".build"
	// ctx, err := experimental.WithCompilationCacheDirName(context.Background(), compilationCacheDir)

	// TODO:  can we use 'ctx'?
	r := wazero.NewRuntimeWithConfig(context.TODO(), f.Config)

	server := RuntimeServer{Runtime: r}
	client := wasm.Runtime_ServerToClient(server)
	return Runtime(client)
}

type Runtime wasm.Runtime

func (r Runtime) AddRef() Runtime {
	return Runtime(wasm.Runtime(r).AddRef())
}

func (r Runtime) Release() {
	wasm.Runtime(r).Release()
}

func (r Runtime) Exec(ctx context.Context, config ConfigFunc) (Proc, capnp.ReleaseFunc) {
	f, release := wasm.Runtime(r).Exec(ctx, config)
	return Proc(f.Proc()), release
}

// func (r Runtime) Close(ctx context.Context) error {
// 	return r.CloseWithStatus(ctx, 0)
// }

// func (r Runtime) CloseWithStatus(ctx context.Context, code uint32) error {
// 	f, release := wasm.Runtime(r).Close(ctx, func(r wasm.Runtime_close_Params) error {
// 		r.SetExitCode(code)
// 		return nil
// 	})
// 	defer release()

// 	return casm.Future(f).Err()
// }

type RuntimeServer struct {
	wazero.Runtime
}

// func (s RuntimeServer) Shutdown() {
// 	if err := s.Runtime.Close(context.TODO()); err != nil {
// 		panic(err)
// 	}
// }

// func (s RuntimeServer) Close(ctx context.Context, call wasm.Runtime_close) error {
// 	return s.Runtime.CloseWithExitCode(ctx, call.Args().ExitCode())
// }

func (s RuntimeServer) Exec(_ context.Context, call api.Executor_exec) error {
	mod, err := s.compile(call)
	if err != nil {
		return fmt.Errorf("module: %w", err)
	}

	return s.run(ctx, mod, config(call))
}

func (s RuntimeServer) compile(call wasm.Runtime_exec) (wazero.CompiledModule, error) {
	src, err := call.Args().Source()
	if err != nil {
		return nil, err
	}

	return s.Runtime.CompileModule(context.TODO(), src)
}

func (s RuntimeServer) run(ctx context.Context, mod wazero.CompiledModule, config wazero.ModuleConfig) error {
	err := gojs.Run(context.TODO(), s.Runtime, mod, config)
	if ex, ok := err.(*sys.ExitError); ok && ex.ExitCode() != 0 {
		return fmt.Errorf("exit(%d)", ex.ExitCode())
	}

	return nil
}

func config(call wasm.Runtime_exec) wazero.ModuleConfig {
	// I/O streams are discarded by default.
	return wazero.NewModuleConfig().
		WithStdin(stdin(call)).
		WithStdout(stdout(call)).
		WithStderr(stderr(call))
}

func stdin(call wasm.Runtime_exec) io.Reader {
	return os.Stdin
}

func stdout(call wasm.Runtime_exec) io.Writer {
	return os.Stdout
}

func stderr(call wasm.Runtime_exec) io.Writer {
	return os.Stderr
}
