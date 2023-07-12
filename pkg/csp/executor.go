package csp

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"lukechampine.com/blake3"

	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"
	api "github.com/wetware/ww/api/process"
	tools_api "github.com/wetware/ww/experiments/api/tools"
	"github.com/wetware/ww/experiments/pkg/tools"
)

// ByteCode is a representation of arbitrary executable data.
type ByteCode []byte

func (b ByteCode) String() string {
	hash := b.Hash()
	return hex.EncodeToString(hash[:])
}

// Hash returns the BLAKE3-256 hash of the byte code.  It is
// suitbale for use as a secure checksum.
func (b ByteCode) Hash() [32]byte {
	return blake3.Sum256(b)
}

// Executor is a capability that can spawn processes.
type Executor api.Executor

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

func (ex Executor) Exec(ctx context.Context, src []byte) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).Exec(ctx, func(ps api.Executor_exec_Params) error {
		return ps.SetBytecode(src)
	})
	return Proc(f.Process()), release
}

func (ex Executor) ExecWithCap(ctx context.Context, src []byte, cap capnp.Client) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).ExecWithCap(ctx, func(ps api.Executor_execWithCap_Params) error {
		if err := ps.SetBytecode(src); err != nil {
			return err
		}
		return ps.SetCap(cap)
	})
	return Proc(f.Process()), release
}

// Server is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Server panics.
type Server struct {
	Runtime wazero.Runtime
}

// Executor provides the Executor capability.
func (r Server) Executor() Executor {
	return Executor(api.Executor_ServerToClient(r))
}

func (r Server) Exec(ctx context.Context, call api.Executor_exec) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	p, err := r.mkproc(ctx, bc, nil)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) ExecWithCap(ctx context.Context, call api.Executor_execWithCap) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	cap := call.Args().Cap().AddRef()

	p, err := r.mkproc(ctx, bc, &cap)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) mkproc(ctx context.Context, bytecode []byte, cap *capnp.Client) (*process, error) {
	mod, err := r.mkmod(ctx, bytecode, cap)
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

func (r Server) mkmod(ctx context.Context, bytecode []byte, cap *capnp.Client) (wasm.Module, error) {
	name := ByteCode(bytecode).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, bytecode)
	if err != nil {
		return nil, err
	}

	// TODO(perf): find a way of locating a free port without opening and closing a connection
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}
	addr := l.Addr().(*net.TCPAddr)

	sockCfg := sock.NewConfig().WithTCPListener("", addr.Port)
	sockCtx := sock.WithConfig(ctx, sockCfg)
	modCfg := wazero.NewModuleConfig().
		WithStartFunctions(). // don't call _start until later
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithName(name).
		WithEnv("ns", name).
		WithStdin(os.Stdin).
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	l.Close()
	mod, err := r.Runtime.InstantiateModule(sockCtx, compiled, modCfg)
	if err != nil {
		return nil, err
	}

	go func() {
		tcpConn, err := dialWithRetries(addr)
		if err != nil {
			panic(err)
		}
		defer tcpConn.Close()

		var client capnp.Client
		// TODO we are always passing a capability, change this so it is not required
		if cap == nil {
			client = capnp.Client(api.Executor_ServerToClient(r))
		} else {
			client = *cap
		}

		inbox := inboxServer{
			Content: client.AddRef(),
		}
		inboxClient := capnp.Client(api.Inbox_ServerToClient(inbox))
		defer inboxClient.Release()
		conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
			BootstrapClient: inboxClient,
			ErrorReporter: errLogger{
				Logger: log.New().WithField("conn", "host"),
			},
		})
		defer conn.Close()

		select {
		case <-conn.Done(): // conn is closed by authenticate if auth fails
			// case <-ctx.Done(): // close conn if the program is exiting
			// TODO ctx.Done is called prematurely when using cluster run
		}
	}()

	return mod, nil
}

func (r Server) spawn(fn wasm.Function) (<-chan execResult, context.CancelFunc) {
	done := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(done)
		defer cancel()

		vs, err := fn.Call(ctx)
		// fmt.Println(err)
		done <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return done, cancel
}

type errLogger struct {
	log.Logger
}

func (e errLogger) ReportError(err error) {
	if err != nil {
		e.WithError(err).Warn("rpc connection failed")
	}
}

// TODO (perf) find a cleaner way
func dialWithRetries(addr *net.TCPAddr) (net.Conn, error) {
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

func (r Server) Tools(ctx context.Context, call api.Executor_tools) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetTools(tools_api.Tools_ServerToClient(tools.ToolServer{}))
}
