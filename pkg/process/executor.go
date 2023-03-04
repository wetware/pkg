package process

import (
	"context"
	"encoding/hex"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/tetratelabs/wazero"
	"lukechampine.com/blake3"

	api "github.com/wetware/ww/internal/api/process"
)

// ByteCode is a representation of arbitrary executable data.
type ByteCode []byte

// String prints the BLAKE3-256 hash of the byte code.  It is
// suitable for use as a secure checksum.
func (b ByteCode) String() string {
	hash := blake3.Sum256(b)
	return hex.EncodeToString(hash[:])
}

type Config struct {
	Executable ByteCode
	EntryPoint string
}

func (c Config) bind(ps api.Executor_spawn_Params) (err error) {
	if err = ps.SetByteCode(c.Executable); err == nil {
		err = ps.SetEntryPoint(c.EntryPoint)
	}

	return
}

// Executor is a capability that can spawn processes.
type Executor api.Executor

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

func (ex Executor) Spawn(ctx context.Context, c Config) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).Spawn(ctx, c.bind)
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

	p, err := wx.mkproc(call.Args())
	if err == nil {
		res.SetProcess(api.Process_ServerToClient(p))
	}

	return err
}

func (wx Server) mkproc(args api.Executor_spawn_Params) (*process, error) {
	bytecode, err := args.ByteCode()
	if err != nil {
		return nil, err
	}

	entrypoint, err := args.EntryPoint()
	if err != nil {
		return nil, err
	}

	return &process{
		Runtime:   wx.Runtime,
		ByteCode:  bytecode,
		EntryFunc: entrypoint,
	}, nil
}
