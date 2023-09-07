package csp_server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"log/slog"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/ipfs/go-cid"
	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"

	core_api "github.com/wetware/pkg/api/core"
	proc_api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/csp/proc"
	"github.com/wetware/pkg/system"
)

// components the Runtime requires to build a process.
type components struct {
	args     csp.Args
	bytecode []byte
	session  core_api.Session
}

type execArgs interface {
	Args() (capnp.TextList, error)
	Ppid() uint32
	Session() (core_api.Session, error)
}

type execRes interface {
	SetProcess(v proc_api.Process) error
}

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
	return csp.Executor(core_api.Executor_ServerToClient(r))
}

func (r Runtime) Exec(ctx context.Context, call core_api.Executor_exec) error {
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

	return r.exec(ctx, cid, bc, call.Args(), res)
}

func (r Runtime) ExecCached(ctx context.Context, call core_api.Executor_execCached) error {
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

	return r.exec(ctx, cid, bc, call.Args(), res)
}

func (r Runtime) exec(ctx context.Context, id cid.Cid, bc []byte, ea execArgs, er execRes) error {
	sess, err := ea.Session()
	if err != nil {
		return err
	}

	args := csp.Args{
		Ppid: r.Tree.PpidOrInit(ea.Ppid()),
		Cid:  id,
		Pid:  r.Tree.NextPid(),
	}

	c := components{
		args:     args,
		bytecode: bc,
		session:  sess,
	}

	p, err := r.mkproc(ctx, c)
	if err != nil {
		return err
	}

	return er.SetProcess(proc_api.Process_ServerToClient(p))
}

func (r Runtime) mkproc(ctx context.Context, c components) (*process, error) {
	mod, err := r.mkmod(ctx, c)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	proc := r.spawn(fn, c.args.Pid, c.args.Ppid)

	return proc, nil
}

func (r Runtime) mkmod(ctx context.Context, c components) (wasm.Module, error) {
	name := csp.ByteCode(c.bytecode).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, c.bytecode)
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
		WithStderr(os.Stderr).
		WithArgs(c.args.Encode()...)

	l.Close()
	mod, err := r.Runtime.InstantiateModule(sockCtx, compiled, modCfg)
	if err != nil {
		return nil, err
	}

	go ServeModule(addr, auth.Session(c.session))

	return mod, nil
}

func (r Runtime) spawn(fn wasm.Function, pid uint32, ppid uint32) *process {
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

	// Register new process.
	r.Tree.Insert(pid, ppid)
	r.Tree.AddToMap(pid, proc)

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

// ServeModule ensures the host side of the TCP connection with addr=addr
// used for CAPNP RPCs is provided by client.
func ServeModule(addr *net.TCPAddr, sess auth.Session) {
	tcpConn, err := DialWithRetries(addr)
	if err != nil {
		panic(err)
	}
	defer tcpConn.Close()

	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		BootstrapClient: capnp.NewClient(core_api.Terminal_NewServer(sess)),
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
