package process

import (
	"context"
	"encoding/hex"
	"fmt"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	"lukechampine.com/blake3"

	wasm "github.com/tetratelabs/wazero/api"
	api "github.com/wetware/ww/internal/api/process"
)

// ByteCode is a representation of arbitrary executable data.
type ByteCode []byte

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

func (ex Executor) Spawn(ctx context.Context, src []byte) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).Spawn(ctx, func(ps api.Executor_spawn_Params) error {
		return ps.SetByteCode(src)
	})
	return Proc(f.Process()), release
}

// Server is the main Executor implementation.  It spawns WebAssembly-
// based processes.  The zero-value Server panics.
type Server struct {
	Runtime wazero.Runtime
}

// Executor provides the Executor capability.
func (wx Server) Executor() Executor {
	return Executor(api.Executor_ServerToClient(wx))
}

// Spawn a process by creating a process server and converting it into
// a capability as a response to the call.
func (wx Server) Spawn(ctx context.Context, call api.Executor_spawn) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	mod, err := wx.loadModule(ctx, call.Args())
	if err != nil {
		return err
	}

	p, err := wx.mkproc(ctx, mod, call.Args())
	if err == nil {
		err = res.SetProcess(api.Process_ServerToClient(p))
	}

	return err
}

func (wx Server) mkproc(ctx context.Context, mod wasm.Module, args api.Executor_spawn_Params) (*process, error) {
	name, err := args.EntryPoint()
	if err != nil {
		return nil, err
	}

	var proc process
	if proc.fn = mod.ExportedFunction(name); proc.fn == nil {
		err = fmt.Errorf("module %s: %s not found", mod.Name(), name)
	}

	return &proc, err
}

func (wx Server) loadModule(ctx context.Context, args api.Executor_spawn_Params) (wasm.Module, error) {
	bc, err := args.ByteCode()
	if err != nil {
		return nil, err
	}

	hash := ByteCode(bc).Hash()
	name := hex.EncodeToString(hash[:])

	config := wazero.
		NewModuleConfig().
		WithName(name)

	if mod := wx.Runtime.Module(name); mod != nil {
		return mod, nil
	}

	module, err := wx.Runtime.CompileModule(ctx, bc)
	if err != nil {
		return nil, err
	}

	return wx.Runtime.InstantiateModule(ctx, module, config)
}
