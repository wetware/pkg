package csp_server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/ipfs/go-cid"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"
	"golang.org/x/exp/slog"

	api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/csp/proc"
	"github.com/wetware/pkg/system"
)

// Runtime is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Runtime panics.
type Runtime struct {
	Runtime wazero.Runtime
	Cache   BytecodeCache
	Tree    ProcTree

	// HostModule is unused for now.
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

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	// Cache new bytecodes every time they are received.
	cid := r.Cache.put(bc)

	var bCtx api.BootContext
	if call.Args().HasBctx() {
		bCtx = call.Args().Bctx()
	} else {
		bCtx = csp.NewBootContext().Cap()
	}
	if err = csp.BootCtx(bCtx).SetCid(ctx, cid); err != nil {
		return err
	}

	ppid := r.Tree.PpidOrInit(call.Args().Ppid())
	pArgs := procArgs{
		bc:   bc,
		ppid: ppid,
		bCtx: bCtx,
	}

	p, err := r.mkproc(ctx, pArgs)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Runtime) ExecCached(ctx context.Context, call api.Executor_execCached) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	b, err := call.Args().Cid()
	if err != nil {
		return err
	}
	_, cid, err := cid.CidFromBytes(b)
	if err != nil {
		return err
	}

	bc := r.Cache.get(cid)
	if bc == nil {
		return fmt.Errorf("bytecode for cid %s not found", cid)
	}

	var bCtx api.BootContext
	if call.Args().HasBctx() {
		bCtx = call.Args().Bctx()
	} else {
		bCtx = csp.NewBootContext().Cap()
	}
	if err = csp.BootCtx(bCtx).SetCid(ctx, cid); err != nil {
		return err
	}

	ppid := r.Tree.PpidOrInit(call.Args().Ppid())
	pArgs := procArgs{
		bc:   bc,
		ppid: ppid,
		bCtx: bCtx,
	}

	p, err := r.mkproc(ctx, pArgs)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Runtime) mkproc(ctx context.Context, args procArgs) (*process, error) {
	pid := r.Tree.NextPid()
	if err := csp.BootCtx(args.bCtx).SetPid(ctx, pid); err != nil {
		return nil, err
	}

	mod, err := r.mkmod(ctx, args)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	proc := r.spawn(fn, pid)

	// Register new process.
	r.Tree.Insert(proc.pid, args.ppid)
	r.Tree.AddToMap(proc.pid, proc)

	return proc, nil
}

func (r Runtime) mkmod(ctx context.Context, args procArgs) (wasm.Module, error) {
	name := csp.ByteCode(args.bc).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, args.bc)
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

	l.Close()
	mod, err := r.Runtime.InstantiateModule(sockCtx, compiled, modCfg)
	if err != nil {
		return nil, err
	}

	go ServeModule(addr, args.bCtx)

	return mod, nil
}

func (r Runtime) spawn(fn wasm.Function, pid uint32) *process {
	done := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	killFunc := r.Tree.Kill
	proc := &process{
		pid:      pid,
		killFunc: killFunc,
		done:     done,
		cancel:   cancel,
	}

	go func() {
		defer close(done)
		defer proc.killFunc(proc.pid)

		vs, err := fn.Call(ctx)

		done <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return proc
}

type procArgs struct {
	bc   []byte
	ppid uint32
	bCtx api.BootContext
}

// ServeModule ensures the host side of the TCP connection with addr=addr
// used for CAPNP RPCs is provided by client.
func ServeModule[T ~capnp.ClientKind](addr *net.TCPAddr, t T) {
	tcpConn, err := DialWithRetries(addr)
	if err != nil {
		panic(err)
	}
	defer tcpConn.Close()

	defer capnp.Client(t).Release()
	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		BootstrapClient: capnp.Client(t),
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default(),
		},
	})
	defer conn.Close()

	select {
	case <-conn.Done(): // conn is closed by authenticate if auth fails
		// case <-ctx.Done(): // close conn if the program is exiting
		// TODO ctx.Done is called prematurely when using cluster run
		// we should use a new context that cancels when subproc ends
	}
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
