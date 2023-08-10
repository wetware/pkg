package csp

import (
	"context"
	"encoding/hex"

	capnp "capnproto.org/go/capnp/v3"
	"lukechampine.com/blake3"

	api "github.com/wetware/pkg/api/process"
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
