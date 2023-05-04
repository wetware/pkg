package csp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/lthibault/log"
	"github.com/tetratelabs/wazero"
	"lukechampine.com/blake3"

	wasm "github.com/tetratelabs/wazero/api"
	cluster_api "github.com/wetware/ww/internal/api/cluster"
	api "github.com/wetware/ww/internal/api/process"
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

// Runtime is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Runtime panics.
type Runtime struct {
	Runtime wazero.Runtime
	Logger  log.Logger
}

// Executor provides the Executor capability.
func (r *Runtime) Executor(host cluster_api.Host) Executor {
	return Executor(api.Executor_ServerToClient(r))
}

func (r *Runtime) Exec(ctx context.Context, call api.Executor_exec) error {
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

func (r *Runtime) mkproc(ctx context.Context, args api.Executor_exec_Params) (*process, error) {
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

func (r *Runtime) mkmod(ctx context.Context, args api.Executor_exec_Params) (wasm.Module, error) {
	bc, err := args.Bytecode()
	if err != nil {
		return nil, err
	}

	name := ByteCode(bc).String()

	// TODO(perf):  cache compiled modules so that we can instantiate module
	//              instances for concurrent use.
	module, err := r.Runtime.CompileModule(ctx, bc)
	if err != nil {
		return nil, err
	}

	host, guest := net.Pipe()

	// TODO wrap guest in host in File, provided by a FS
	// TODO wrap host in RPC.NewStreamTransport(?) and provide the bootstrap capability
	// in separate goroutine

	return r.Runtime.InstantiateModule(ctx, module, wazero.
		NewModuleConfig().
		WithName(name).
		WithStartFunctions(). // disable automatic calling of _start (main)
		WithRandSource(rand.Reader).
		WithFS(FS{conn: guest}))
}

func (r *Runtime) spawn(fn wasm.Function) (<-chan execResult, context.CancelFunc) {
	out := make(chan execResult, 1)

	// NOTE:  we use context.Background instead of the context obtained from the
	//        rpc handler. This ensures that a process can continue to run after
	//        the rpc handler has returned. Note also that this context is bound
	//        to the application lifetime, so processes cannot block a shutdown.
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(out)
		defer cancel()

		vs, err := fn.Call(ctx)
		out <- execResult{
			Values: vs,
			Err:    err,
		}
	}()

	return out, cancel
}
