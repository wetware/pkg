package csp_server

import (
	"context"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"time"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"
	"golang.org/x/exp/slog"

	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/csp/proc"
	"github.com/wetware/pkg/util/log"
)

// Runtime is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Runtime panics.
type Runtime struct {
	Runtime    wazero.Runtime
	HostModule *wazergo.ModuleInstance[*proc.Module]
}

// Executor provides the Executor capability.
func (r Runtime) Executor() csp.Executor {
	return csp.Executor(api.Executor_ServerToClient(r))
}

func (r Runtime) Exec(ctx context.Context, call api.Executor_exec) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	p, err := r.mkproc(ctx, call.Args())
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Runtime) mkproc(ctx context.Context, args api.Executor_exec_Params) (*process, error) {
	mod, err := r.mkmod(ctx, args)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	done, cancel := r.spawn(fn)
	return &process{
		done:   done,
		cancel: cancel,
	}, nil
}

func (r Runtime) mkmod(ctx context.Context, args api.Executor_exec_Params) (wasm.Module, error) {
	bc, err := args.Bytecode()
	if err != nil {
		return nil, err
	}

	name := csp.ByteCode(bc).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, bc)
	if err != nil {
		return nil, err
	}

	// TODO(perf): find a way of locating a free port without opening and
	//             closing a connection.
	// Find a free TCP port.
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)

	// Enables the creation of non-blocking TCP connections
	// inside the WASM module. The host will pre-open the TCP
	// port and pass it to the guest through a file descriptor.
	sockCfg := sock.NewConfig().WithTCPListener("", addr.Port)
	sockCtx := sock.WithConfig(ctx, sockCfg)
	modCfg := wazero.NewModuleConfig().
		WithStartFunctions(). // don't call _start until later
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithRandSource(rand.Reader).
		WithName(name).
		WithEnv("ns", name).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)
	mod, err := r.Runtime.InstantiateModule(sockCtx, compiled, modCfg)
	if err != nil {
		return nil, err
	}

	raw, err := DialWithRetries(addr)
	if err != nil {
		panic(err)
	}
	conn := rpc.NewConn(rpc.NewStreamTransport(raw), &rpc.Options{
		BootstrapClient: args.BootstrapClient(),
		ErrorReporter: &log.ErrorReporter{
			Logger: slog.Default(),
		},
	})
	defer raw.Close()

	go func() {
		defer conn.Close()

		select {
		case <-conn.Done(): // conn is closed by authenticate if auth fails
			// case <-ctx.Done(): // close conn if the program is exiting
			// TODO ctx.Done is called prematurely when using cluster run
			// we should use a new context that cancels when subproc ends
		}
	}()

	return mod, nil
}

func (r Runtime) spawn(fn wasm.Function) (<-chan execResult, context.CancelFunc) {
	out := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(out)
		defer cancel()

		vs, err := fn.Call(wazergo.WithModuleInstance(ctx, r.HostModule))
		out <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return out, cancel
}

// DialWithRetries dials addr in waitTime intervals until it either succeeds or
// exceeds maxRetries retries.
func DialWithRetries(addr *net.TCPAddr) (net.Conn, error) {
	maxRetries := 20
	waitTime := 10 * time.Millisecond
	var err error
	var conn net.Conn

	for retries := 0; retries < maxRetries; retries++ {
		conn, err = net.Dial("tcp", addr.String())
		if err == nil {
			break
		}
		time.Sleep(waitTime)
	}

	return conn, err
}
