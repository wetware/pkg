package wasm

import (
	"context"
	"fmt"
	"sync"

	"capnproto.org/go/capnp/v3"

	"github.com/tetratelabs/wazero"
	gojs "github.com/tetratelabs/wazero/imports/go"
	"github.com/tetratelabs/wazero/sys"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/ww/internal/api/proc"
	"github.com/wetware/ww/internal/api/wasm"
	"github.com/wetware/ww/pkg/process"
)

type Param process.Param[wasm.Runtime_Config]

type RunContext process.Config[wasm.Runtime_Config]

func NewRunContext(src []byte) RunContext {
	config := process.NewConfig(wasm.NewRuntime_Config)
	return RunContext(config).Bind(source(src))
}

func source(b []byte) Param {
	return func(rc wasm.Runtime_Config) error {
		return rc.SetSrc(b)
	}
}

func (c RunContext) Bind(p Param) RunContext {
	param := process.Param[wasm.Runtime_Config](p)
	config := process.Config[wasm.Runtime_Config](c)
	return RunContext(config.Bind(param))
}

func (c RunContext) WithEnv(env map[string]string) RunContext {
	return c.Bind(func(cr wasm.Runtime_Config) error {
		fs, err := cr.NewEnv(int32(len(env)))
		if err != nil {
			return err
		}

		var i int
		for k, v := range env {
			if err = fs.At(i).SetKey(k); err != nil {
				break
			}

			if err = fs.At(i).SetValue(v); err != nil {
				break
			}
		}

		return err
	})
}

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

func (r Runtime) Exec(ctx context.Context, c RunContext) (Proc, capnp.ReleaseFunc) {
	f, release := wasm.Runtime(r).Exec(ctx, c)
	return Proc(f.Proc()), release
}

type Proc wasm.Runtime_Context

func (p Proc) AddRef() Proc {
	return Proc(capnp.Client(p).AddRef())
}

func (p Proc) Release() {
	capnp.Client(p).Release()
}

func (p Proc) Run(ctx context.Context) (casm.Future, capnp.ReleaseFunc) {
	f, release := wasm.Runtime_Context(p).Run(ctx, nil)
	return casm.Future(f), release
}

func (p Proc) Wait(ctx context.Context) error {
	f, release := wasm.Runtime_Context(p).Wait(ctx, nil)
	defer release()

	return casm.Future(f).Err()
}

func (p Proc) Close(ctx context.Context) error {
	return p.CloseWithExitCode(ctx, 0)
}

func (p Proc) CloseWithExitCode(ctx context.Context, status uint32) error {
	f, release := wasm.Runtime_Context(p).Close(ctx, statusCode(status))
	defer release()

	return casm.Future(f).Err()
}

func statusCode(u uint32) func(wasm.Runtime_Context_close_Params) error {
	return func(ps wasm.Runtime_Context_close_Params) error {
		ps.SetExitCode(u)
		return nil
	}
}

type RuntimeServer struct {
	wazero.Runtime
}

func (s RuntimeServer) Exec(_ context.Context, call proc.Executor_exec) error {
	mod, err := s.compile(call)
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	cfg, err := s.config(call)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	rx := wasm.Runtime_Context_ServerToClient(&execContext{
		Runtime: s.Runtime,
		Module:  mod,
		Config:  cfg,
		done:    make(chan struct{}),
	})

	res, err := call.AllocResults()
	if err == nil {
		err = res.SetProc(proc.Waiter(rx))
	}

	return err
}

func (s RuntimeServer) compile(call proc.Executor_exec) (wazero.CompiledModule, error) {
	c, err := config(call)
	if err != nil {
		return nil, err
	}

	src, err := c.Src()
	if err != nil {
		return nil, err
	}

	return s.Runtime.CompileModule(context.TODO(), src)
}

func (s RuntimeServer) config(call proc.Executor_exec) (wazero.ModuleConfig, error) {
	c, err := config(call)
	if err != nil {
		return nil, err
	}

	conf := wazero.NewModuleConfig()

	// I/O streams are discarded by default.
	conf = conf.
		WithStdin(stdin(c)).
		WithStdout(stdout(c)).
		WithStderr(stderr(c))

	// Set the environment
	env, err := c.Env()
	if err != nil {
		return nil, err
	}

	for i := 0; i < env.Len(); i++ {
		key, err := env.At(i).Key()
		if err != nil {
			return nil, err
		}

		val, err := env.At(i).Value()
		if err != nil {
			return nil, err
		}

		conf = conf.WithEnv(key, val)
	}

	return conf, nil
}

func config(call proc.Executor_exec) (wasm.Runtime_Config, error) {
	ptr, err := call.Args().Config()
	return wasm.Runtime_Config(ptr.Struct()), err
}

type execContext struct {
	Runtime wazero.Runtime
	Module  wazero.CompiledModule
	Config  wazero.ModuleConfig

	once sync.Once
	stat *sys.ExitError
	done chan struct{}
}

func (ex *execContext) Shutdown() {
	ex.once.Do(func() {
		close(ex.done)
	})
}

func (ex *execContext) Run(ctx context.Context, call wasm.Runtime_Context_run) error {
	ex.once.Do(func() {
		defer close(ex.done)
		call.Ack()

		err := gojs.Run(context.TODO(), ex.Runtime, ex.Module, ex.Config)
		if e, ok := err.(*sys.ExitError); ok && e.ExitCode() != 0 {
			ex.stat = e
		}
	})

	if ex.stat != nil {
		return ex.stat
	}

	return nil
}

func (ex *execContext) Close(ctx context.Context, call wasm.Runtime_Context_close) error {
	if status := call.Args().ExitCode(); status != 0 {
		return ex.Runtime.CloseWithExitCode(ctx, status)
	}

	return ex.Runtime.Close(ctx)
}

func (ex *execContext) Wait(ctx context.Context, call proc.Waiter_wait) error {
	select {
	case <-ex.done:
		res, err := call.AllocResults()
		if err != nil {
			return err
		}

		stat, err := wasm.NewRuntime_Context_Status(res.Segment())
		if err == nil {
			stat.SetStatusCode(ex.stat.ExitCode())
		}

		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}
