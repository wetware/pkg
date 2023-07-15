package server

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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
	"github.com/stealthrocket/wzprof"
	"github.com/tetratelabs/wazero"
	"lukechampine.com/blake3"

	wasm "github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
	"github.com/tetratelabs/wazero/experimental/sock"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	api "github.com/wetware/ww/api/process"
	tools_api "github.com/wetware/ww/experiments/api/tools"
	"github.com/wetware/ww/experiments/pkg/tools"
	csp "github.com/wetware/ww/pkg/csp"
)

// ByteCode is a representation of arbitrary executable data.
type ByteCode []byte

func (b ByteCode) String() string {
	hash := b.Hash()
	return hex.EncodeToString(hash[:])
}

// Hash returns the BLAKE3-256 hash of the byte code. It is
// suitbale for use as a secure checksum.
func (b ByteCode) Hash() [32]byte {
	return blake3.Sum256(b)
}

// Server is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Server panics.
type Server struct {
	Profile    bool
	Runtime    wazero.Runtime
	BcRegistry RegistryServer
	ProcTree   ProcTree

	profileruntimeset bool
}

// Executor provides the Executor capability.
func (r Server) Executor() csp.Executor {
	return csp.Executor(api.Executor_ServerToClient(r))
}

func (r Server) Exec(ctx context.Context, call api.Executor_exec) error {
	if r.Profile {
		cpuProfile, _ := os.Create("cpuprofile_exec.prof")
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

	// Check for a process with pid=ppid.
	ppid := call.Args().Ppid()
	if ppid == 0 {
		ppid = INIT_PID
	} else {
		if _, ok := r.ProcTree.Map[ppid]; !ok {
			return fmt.Errorf("pid %d not found", ppid)
		}
	}

	// Cache new bytecodes by registering them every time they are received.
	r.BcRegistry.put(bc)

	// Prepare the capability list that will be passed downstream.
	// If call.Caps will be used if non-null, otherwise an empty list
	// will be used instead.
	var caps capnp.PointerList
	if call.Args().HasCaps() {
		caps, err = call.Args().Caps()
		if err != nil {
			return err
		}
	} else {
		_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return err
		}
		caps, err = capnp.NewPointerList(seg, 0)
		if err != nil {
			return err
		}
	}

	p, err := r.mkproc(ctx, ppid, bc, caps)
	if err != nil {
		return err
	}

	// Register new pid.
	r.ProcTree.Map[p.pid] = p

	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) ExecFromCache(ctx context.Context, call api.Executor_execFromCache) error {
	ppid := call.Args().Ppid()
	if r.Profile && ppid == 1 {
		cpuProfile, _ := os.Create(fmt.Sprintf("cpuprofile_execwithcap.%d.prof", ppid))
		pprof.StartCPUProfile(cpuProfile)
		defer pprof.StopCPUProfile()
	} else if r.Profile && ppid > 1 {
		return errors.New("stop at first profiled proc")
	}
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	caps, err := call.Args().Caps()
	if err != nil {
		return err
	}

	if ppid == 0 {
		ppid = INIT_PID
	} else {
		if _, ok := r.ProcTree.Map[ppid]; !ok {
			return fmt.Errorf("pid %d not found", ppid)
		}
	}

	md5sum, err := call.Args().Md5sum()
	if err != nil {
		return err
	}

	if len(md5sum) != md5.Size {
		return fmt.Errorf("unexpected md5sum size, got %d expected %d", len(md5sum), md5.Size)
	}

	bc := r.BcRegistry.get(md5sum)
	if bc == nil {
		return fmt.Errorf("bytecode for md5 sum %s not found", md5sum)
	}

	p, err := r.mkproc(ctx, ppid, bc, caps)
	if err != nil {
		return err
	}

	// Register new pid.
	r.ProcTree.Map[p.pid] = p
	return res.SetProcess(api.Process_ServerToClient(p))
}

func (r Server) mkproc(ctx context.Context, ppid uint32, bytecode []byte, caps capnp.PointerList) (*process, error) {
	pid := r.ProcTree.PIDC.Inc()

	mod, cpuProf, err := r.mkmod(ctx, bytecode, pid, caps)
	if err != nil {
		return nil, err
	}

	fn := mod.ExportedFunction("_start")
	if fn == nil {
		return nil, errors.New("ww: missing export: _start")
	}

	proc := r.spawn(fn, pid, cpuProf)
	r.ProcTree.Insert(pid, ppid)

	return proc, nil
}

func (r Server) mkmod(ctx context.Context, bytecode []byte, pid uint32, caps capnp.PointerList) (wasm.Module, *wzprof.CPUProfiler, error) {
	name := fmt.Sprintf(
		"%s-%s",
		ByteCode(bytecode).String(), // TODO standardize hashes, md5...
		uuid.New(),
	)

	// profiling variables
	var p *wzprof.Profiling
	var cpuProf *wzprof.CPUProfiler
	var pprofCtx context.Context

	// set profiling runtime
	if r.Profile {
		p = wzprof.ProfilingFor(bytecode)
		cpuProf = p.CPUProfiler()
		pprofCtx = context.WithValue(context.Background(),
			experimental.FunctionListenerFactoryKey{},
			experimental.MultiFunctionListenerFactory(
				wzprof.Sample(1.0, cpuProf),
			))
		ctx = pprofCtx
		if !r.profileruntimeset {
			runtimeCfg := wazero.
				NewRuntimeConfigCompiler().
				WithCompilationCache(wazero.NewCompilationCache())
			r.Runtime = wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
			_, err := wasi_snapshot_preview1.Instantiate(ctx, r.Runtime)
			if err != nil {
				panic(err)
			}
			r.profileruntimeset = true
		}
	}

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	compiled, err := r.Runtime.CompileModule(ctx, bytecode)
	if err != nil {
		return nil, nil, err
	}

	if r.Profile {
		if err = p.Prepare(compiled); err != nil {
			panic(err)
		}
		cpuProf.StartProfile()
	}

	// TODO(perf): find a way of locating a free port without opening and closing a connection
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	// TODO the private key is being sent unencrypted over the wire.
	// Send it over an encrypted channel instead.
	md5sum := md5.Sum(bytecode)
	context, err := r.populateContext(pid, md5sum[:], caps)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		tcpConn, err := dialWithRetries(addr)
		if err != nil {
			panic(err)
		}
		defer tcpConn.Close()

		defer context.Release()
		conn := rpc.NewConn(rpc.NewStreamTransport(tcpConn), &rpc.Options{
			BootstrapClient: context,
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
	}()

	return mod, cpuProf, nil
}

func (r Server) populateContext(pid uint32, md5sum []byte, caps capnp.PointerList) (capnp.Client, error) {
	var context anyIbox
	var err error

	// Args that will be present in all processes.
	initArgs := csp.NewArgs(
		strconv.FormatUint(uint64(pid), 10),
		string(md5sum),
	)

	// The process is provided its own executor by default.
	if caps.Len() <= 0 {
		executor := capnp.Client(api.Executor_ServerToClient(r))
		context = newDecodedContext(capnp.Client(initArgs), capnp.Client(csp.NewArgs()), executor)
	} else { // Otherwise it will pass the received capabilities.
		context, err = newEncodedContext(caps, capnp.Client(initArgs))
		if err != nil {
			return capnp.Client{}, nil
		}
	}

	return capnp.Client(api.Context_ServerToClient(context)), nil
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
