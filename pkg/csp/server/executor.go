package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/google/uuid"
	"github.com/lthibault/log"
	"github.com/mr-tron/base58/base58"
	"github.com/stealthrocket/wzprof"
	"github.com/tetratelabs/wazero"

	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/sock"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	api "github.com/wetware/ww/api/process"
	tools_api "github.com/wetware/ww/experiments/api/tools"
	"github.com/wetware/ww/experiments/pkg/tools"
	csp "github.com/wetware/ww/pkg/csp"
)

// Server is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Server panics.
type Server struct {
	Profile    bool
	Runtime    wazero.Runtime
	BcRegistry RegistryServer
	ProcTree   ProcTree

	profilingruntimeset bool
}

// Executor provides the Executor capability.
func (r Server) Executor() csp.Executor {
	return csp.Executor(api.Executor_ServerToClient(r))
}

func (r Server) Exec(ctx context.Context, call api.Executor_exec) error {
	ppid := r.ppidOrInit(call.Args().Ppid())
	// Profiling block.
	if r.Profile {
		cpuProfile, _ := os.Create(fmt.Sprintf("cpuprofile_exec.%d.prof", ppid))
		pprof.StartCPUProfile(cpuProfile)
		defer pprof.StopCPUProfile()
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	bc, err := call.Args().Bytecode()
	if err != nil {
		return err
	}

	// Cache new bytecodes by registering them every time they are received.
	r.BcRegistry.put(bc)

	// Prepare the capability list that will be passed downstream.
	// call.Caps will be used if non-null, otherwise an empty list
	// will be used instead.
	caps, err := capsOrEmpty(call.Args().HasCaps, call.Args().Caps)
	if err != nil {
		return err
	}

	p, err := r.mkproc(ctx, ppid, bc, caps)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) ExecFromCache(ctx context.Context, call api.Executor_execFromCache) error {
	ppid := r.ppidOrInit(call.Args().Ppid())
	// Profiling block.
	if r.Profile {
		cpuProfile, _ := os.Create(fmt.Sprintf("cpuprofile_execwithcap.%d.prof", ppid))
		pprof.StartCPUProfile(cpuProfile)
		defer pprof.StopCPUProfile()
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	// Prepare the capability list that will be passed downstream.
	// call.Caps will be used if non-null, otherwise an empty list
	// will be used instead.
	caps, err := capsOrEmpty(call.Args().HasCaps, call.Args().Caps)
	if err != nil {
		return err
	}

	hash, err := call.Args().Hash()
	if err != nil {
		return err
	}

	if len(hash) != csp.HashSize {
		return fmt.Errorf("unexpected hash size, got %d expected %d", len(hash), csp.HashSize)
	}

	bc := r.BcRegistry.get(hash)
	if bc == nil {
		return fmt.Errorf("bytecode for hash %s not found", hash)
	}

	p, err := r.mkproc(ctx, ppid, bc, caps)
	if err != nil {
		return err
	}

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) Registry(ctx context.Context, call api.Executor_registry) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}
	return res.SetRegistry(api.BytecodeRegistry_ServerToClient(r.BcRegistry))
}

func (r Server) Tools(ctx context.Context, call api.Executor_tools) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetTools(tools_api.Tools_ServerToClient(tools.ToolServer{}))
}

// mkproc creates a new process with parent ppid from the specified bytecode
// and puts caps in its boot context.
func (r Server) mkproc(ctx context.Context, ppid uint32, bytecode []byte, caps capnp.PointerList) (*process, error) {
	pid := r.ProcTree.NextPid()

	mod, cpuProf, err := r.mkmod(ctx, bytecode, pid, caps)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	proc := r.spawn(fn, pid, cpuProf)

	// Register new process.
	r.ProcTree.Insert(proc.pid, ppid)
	r.ProcTree.AddToMap(proc.pid, proc)

	return proc, nil
}

// mkmod creates a WASM module from the specified bytecode. The WASM module
// will be binded to PID=pid and will have caps in its boot context.
func (r Server) mkmod(ctx context.Context, bc []byte, pid uint32, caps capnp.PointerList) (wasm.Module, *wzprof.CPUProfiler, error) {
	hash := csp.HashFunc(bc)
	name := fmt.Sprintf(
		"%s-%s",
		base58.FastBase58Encoding(hash[:]),
		uuid.New(),
	)

	// Profiling block.
	var p *wzprof.Profiling
	var cpuProf *wzprof.CPUProfiler
	var pprofCtx context.Context
	if r.Profile {
		p = wzprof.ProfilingFor(bc)
		cpuProf = p.CPUProfiler()
		pprofCtx = context.WithValue(context.Background(),
			experimental.FunctionListenerFactoryKey{},
			experimental.MultiFunctionListenerFactory(
				wzprof.Sample(1.0, cpuProf),
			))
		ctx = pprofCtx
		if !r.profilingruntimeset {
			runtimeCfg := wazero.
				NewRuntimeConfigCompiler().
				WithCompilationCache(wazero.NewCompilationCache()).
				WithCloseOnContextDone(true)
			r.Runtime = wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
			_, err := wasi_snapshot_preview1.Instantiate(ctx, r.Runtime)
			if err != nil {
				panic(err)
			}
			r.profilingruntimeset = true
		}
	}

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, bc)
	if err != nil {
		return nil, nil, err
	}

	// Profiling block.
	if r.Profile {
		if err = p.Prepare(compiled); err != nil {
			panic(err)
		}
		cpuProf.StartProfile()
	}

	// TODO(perf): find a way of locating a free port without opening and
	//             closing a connection.
	// Find a free TCP port.
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}
	addr := l.Addr().(*net.TCPAddr)

	// Enables the creation of non-blocking TCP connections
	// inside the WASM module. The host will pre-open the TCP
	// port and pass it to the guest through a file descriptor.
	sockCfg := sock.NewConfig().WithTCPListener("", addr.Port)
	sockCtx := sock.WithConfig(ctx, sockCfg)
	// Default module configuration.
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
		return nil, nil, err
	}

	bootContext, err := r.populateBootContext(pid, hash[:], caps)
	if err != nil {
		return nil, nil, err
	}

	go serveModule(addr, capnp.Client(bootContext))

	return mod, cpuProf, nil
}

// serveModule ensures the host side of the TCP connection with addr=addr
// used for CAPNP RPCs is provided by client.
func serveModule(addr *net.TCPAddr, client capnp.Client) {
	tcpConn, err := dialWithRetries(addr)
	if err != nil {
		panic(err)
	}
	defer tcpConn.Close()

	defer client.Release()
	conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
		BootstrapClient: client,
		ErrorReporter: errLogger{
			Logger: log.New(log.WithLevel(log.ErrorLevel)).WithField("conn", "host"),
		},
	})
	defer conn.Close()

	select {
	case <-conn.Done(): // conn is closed by authenticate if auth fails
		// case <-ctx.Done(): // close conn if the program is exiting
		// TODO ctx.Done is called prematurely when using cluster run
	}
}

func (r Server) spawn(fn wasm.Function, pid uint32, cpuProf *wzprof.CPUProfiler) *process {
	done := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	killFunc := r.ProcTree.Kill
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

		// Profiling block.
		if cpuProf != nil {
			prof := cpuProf.StopProfile(1.0)
			if werr := wzprof.WriteProfile("wasm.prof", prof); werr != nil {
				defer panic(err)
			}
		}

		done <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return proc
}

// ppidOrInit checks for a process with pid=ppid and returns
// ppid if found, INIT_PID otherwise.
func (r Server) ppidOrInit(ppid uint32) uint32 {
	if ppid == 0 {
		return INIT_PID
	} else {
		// Default INIT_PID as a parent.
		if _, ok := r.ProcTree.Map[ppid]; !ok {
			return INIT_PID
		}
	}
	return ppid
}

// capsOrEmpty extracts the caps from a (exec|execFromCache) call and returns them.
// It will return an empty list if there were none.
func capsOrEmpty(
	hasCaps func() bool,
	getCaps func() (capnp.PointerList, error),
) (capnp.PointerList, error) {
	var caps capnp.PointerList
	var err error
	if hasCaps() {
		caps, err = getCaps()
	} else {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return caps, err
		}
		return capnp.NewPointerList(seg, 0)
	}
	return caps, err
}

// populateBootContext creates a BootContext with the following items:
// 1. Args(pid, hash)
// If caps.Len() == 0:
//  2. Empty Args
//  3. Capability of executor r
//
// Else:
//  2. caps[0]
//  3. caps[1]
//     ...
//     n. caps[n-2]
func (r Server) populateBootContext(pid uint32, hash []byte, caps capnp.PointerList) (api.BootContext, error) {
	var bootContext anyIbox
	var err error

	// Args that will be present in all processes.
	initArgs := csp.NewArgs(
		strconv.FormatUint(uint64(pid), 10),
		string(hash),
	)

	// The process is provided its own executor by default.
	if caps.Len() <= 0 {
		executor := capnp.Client(api.Executor_ServerToClient(r))
		bootContext = newDecodedBootContext(
			capnp.Client(initArgs), capnp.Client(csp.NewArgs()), executor)
	} else { // Otherwise it will pass the received capabilities.
		bootContext, err = newEncodedBootContext(caps, capnp.Client(initArgs))
		if err != nil {
			return api.BootContext{}, nil
		}
	}

	return api.BootContext_ServerToClient(bootContext), nil
}

// dialWithRetries dials addr in waitTime intervals until it either succeeds or
// exceeds maxRetries retries.
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

type errLogger struct {
	log.Logger
}

func (e errLogger) ReportError(err error) {
	if err != nil {
		e.WithError(err).Warn("rpc connection failed")
	}
}
