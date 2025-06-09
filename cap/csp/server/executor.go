package csp_server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"os"
	"sync"
	"time"

	"log/slog"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	capnp_server "capnproto.org/go/capnp/v3/server"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	"github.com/tetratelabs/wazero"
	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental/sock"

	core_api "github.com/wetware/pkg/api/core"
	proc_api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/log"
)

var nilCid, _ = cid.V1Builder{}.Sum([]byte{})

// components the Runtime requires to build a process.
type components struct {
	args      csp.Args
	bootstrap *capnp_server.Server
	bytecode  []byte

	ctx    context.Context
	cancel context.CancelFunc
}

type ExecArgs struct {
	Argv      []string
	Bootstrap *capnp_server.Server
	Bytecode  []byte
	Cid       cid.Cid
	Ppid      uint32
}

// Runtime is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Runtime panics.
type Runtime struct {
	Runtime  wazero.Runtime
	Cache    BytecodeCache
	Tree     ProcTree
	Log      log.Logger
	PeerDial func(context.Context, core_api.Executor_dialPeer) error
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
	cid := r.Cache.put(bc)

	encodedArgs, err := call.Args().Args()
	if err != nil {
		return err
	}

	argv, err := csp.DecodeTextList(encodedArgs)
	if err != nil {
		return nil
	}

	bootstrap, err := cloneBootstrap(ctx, call.Args().Bootstrap())
	if err != nil {
		return err
	}

	ea := ExecArgs{
		Argv:      argv,
		Bootstrap: bootstrap,
		Bytecode:  bc,
		Cid:       cid,
		Ppid:      call.Args().Ppid(),
	}

	p, err := r.exec(ctx, ea)
	if err != nil {
		return err
	}

	return res.SetProcess(p)
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

	encodedArgs, err := call.Args().Args()
	if err != nil {
		return err
	}

	argv, err := csp.DecodeTextList(encodedArgs)
	if err != nil {
		return nil
	}

	bootstrap, err := cloneBootstrap(ctx, call.Args().Bootstrap())
	if err != nil {
		return err
	}

	ea := ExecArgs{
		Argv:      argv,
		Bootstrap: bootstrap,
		Bytecode:  bc,
		Cid:       cid,
		Ppid:      call.Args().Ppid(),
	}

	p, err := r.exec(ctx, ea)

	if err != nil {
		return err
	}

	return res.SetProcess(p)
}

func (r Runtime) ExposedExec(ctx context.Context, ea ExecArgs) (proc_api.Process, error) {
	return r.exec(ctx, ea)
}

// TODO mikel: graceful process shutdown. Currently an expected EOF error appears.
func (r Runtime) exec(ctx context.Context, ea ExecArgs) (proc_api.Process, error) {

	cid := ea.Cid
	if cid == nilCid {
		cid = rom.ROM{Bytecode: ea.Bytecode}.CID()
	}

	args := csp.Args{
		Ppid: r.Tree.PpidOrInit(ea.Ppid),
		Cid:  cid,
		Pid:  r.Tree.NextPid(),
		Cmd:  ea.Argv,
	}
	r.Log.Info("exec",
		"pid", args.Pid,
		"ppid", args.Ppid,
		"cid", cid.Encode(multibase.MustNewEncoder(multibase.Base58BTC)),
		"args", ea.Argv)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	cctx, ccancel := context.WithCancel(context.Background())
	c := components{
		args:      args,
		bytecode:  ea.Bytecode,
		bootstrap: ea.Bootstrap,
		ctx:       cctx,
		cancel:    ccancel,
	}

	p, err := r.mkproc(ctx, c)
	if err != nil {
		return proc_api.Process{}, err
	}

	return proc_api.Process_ServerToClient(p), nil
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

	proc := r.spawn(fn, c)

	return proc, nil
}

func (r Runtime) mkmod(ctx context.Context, c components) (wasm.Module, error) {
	name := csp.ByteCode(c.bytecode).String() + uuid.NewString()

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

	r.Log.Info("instantiate module", "name", name, "port", addr.Port)
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

	r.Log.Info("serve module", "pid", c.args.Pid, "cid", c.args.Cid.String())
	go ServeModule(c.ctx, addr, c.bootstrap)

	return mod, nil
}

func (r Runtime) spawn(fn wasm.Function, c components) *process {
	done := make(chan execResult, 1)

	killFunc := r.Tree.Kill
	proc := &process{
		Args:      c.args,
		time:      time.Now().UnixMilli(),
		killFunc:  killFunc,
		done:      done,
		cancel:    c.cancel,
		procFetch: r.fetchLocalProc,

		id:         mrand.Int63(),
		links:      &sync.Map{},
		localLinks: &sync.Map{},
		monitors:   make(chan proc_api.Process_monitor),
		events:     nilEvents,
	}

	// Register new process.
	r.Tree.Insert(c.args.Pid, c.args.Ppid)
	r.Tree.AddToMap(c.args.Pid, proc)

	go func() {
		defer close(done)
		defer c.cancel()       // stop the rpc provider
		defer proc.kill(c.ctx) // terminate the process
		vs, err := fn.Call(c.ctx)

		done <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return proc
}

// @lthibault: When someone calls Executor.exec, to spawn a process, it sends a proc_api.Bootstrap
// capability as a call argument. I've tried to use that capability directly in `ServeModule` to
// make the original caller serve the `proc_api.Bootstrap` capability, but I  got a
// `VAT does not expose a public/bootstrap interface` error. Creating a new in-process
// `proc_api.Bootstrap` server with all the capabilities of the original, which is what
// `cloneBootstrap` does, solved the issue. Still, it seems a bit redundant. Do you know what
// would be causing the `VAT does not expose a public/bootstrap interface` error?
func cloneBootstrap(ctx context.Context, bs proc_api.Bootstrap) (*capnp_server.Server, error) {
	// Clone the Bootstrap capability into a server to make it provideable
	// to the client. This is required because the client VAT doesn't
	// expose a public/bootstrap interface.
	bootstrap := csp.ProcessBootstrap(bs)
	clone := NewProcessBootstrap()

	for {
		cap, release, hasNext, err := bootstrap.Get(ctx)
		if err != nil {
			return nil, err
		}

		if !hasNext {
			break
		}
		clone.add(cap.AddRef())
		PendingReleases = append(PendingReleases, release)
	}

	return proc_api.Bootstrap_NewServer(clone), nil
}

// ServeModule ensures the host side of the TCP connection with addr=addr
// used for CAPNP RPCs is provided by client.
func ServeModule(ctx context.Context, addr *net.TCPAddr, bootstrap *capnp_server.Server) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// TODO @mikelsr @lthibault this is were modules non-bootstrapping
	// modules fail. Recovering is not an option, I think we'd
	// much rather find the cause and fix it. Still, leaving this here
	// for reference.
	// 	}
	// }()

	tcpConn, err := DialLoop(ctx, addr, 0)
	if err != nil {
		panic(err)
	}
	defer tcpConn.Close()
	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		BootstrapClient: capnp.NewClient(bootstrap),
		Logger: system.ErrorReporter{
			Logger: slog.Default(),
		},
	})
	defer conn.Close()
	select {
	case <-ctx.Done(): // close conn if the program is exiting
		conn.Close()
	case <-conn.Done(): // conn is closed by authenticate if auth fails
	}
}

// DialLoop dials addr in waitTime intervals until it either succeeds or
// the context is cancelled. Set retries to 0 for infinite loop.
func DialLoop(ctx context.Context, addr *net.TCPAddr, retries int) (net.Conn, error) {
	waitTime := 10 * time.Millisecond
	var err error
	var conn net.Conn

	i := 0
	for {
		conn, err = net.Dial("tcp", addr.String())
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(waitTime):
			waitTime *= 2
		}

		if retries != 0 && i >= retries {
			return nil, errors.New("retries exceeded")
		}
	}

	return conn, err
}

// Ps returns the info of every running processes.
func (r Runtime) Ps(ctx context.Context, call core_api.Executor_ps) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	snap := r.Tree.MapSnapshot()
	_, seg := capnp.NewSingleSegmentMessage(nil)
	pl, err := proc_api.NewInfo_List(seg, int32(len(snap)))
	if err != nil {
		return err
	}

	i := 0
	for _, v := range snap {
		info, err := v.(*process).info()
		if err != nil {
			return err
		}
		if err = pl.Set(i, info); err != nil {
			return err
		}
		i++
	}
	return res.SetProcs(pl)
}

func (r Runtime) BytecodeCache(ctx context.Context, call core_api.Executor_bytecodeCache) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetCache(proc_api.BytecodeCache_ServerToClient(r.Cache))
}

func (r Runtime) fetchLocalProc(pid uint32) (*process, bool) {
	p, ok := r.Tree.Map.Load(pid)
	if !ok {
		return nil, ok
	}
	return p.(*process), ok
}

func (r Runtime) DialPeer(ctx context.Context, call core_api.Executor_dialPeer) error {
	call.Go()
	return r.PeerDial(ctx, call)
}
