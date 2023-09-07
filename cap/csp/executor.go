package csp

import (
	"context"
	"encoding/hex"

	capnp "capnproto.org/go/capnp/v3"
	"lukechampine.com/blake3"

	"github.com/ipfs/go-cid"
	core_api "github.com/wetware/pkg/api/core"
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
type Executor core_api.Executor

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

// Exec spawns a new process from WASM bytecode bc. If the caller is a WASM process
// spawned in this same executor, it should use its PID as ppid to mark the
// new process as a subprocess.
// TODO args
func (ex Executor) Exec(ctx context.Context, sess core_api.Session, bc []byte, ppid uint32) (Proc, capnp.ReleaseFunc) {
	f, release := core_api.Executor(ex).Exec(ctx,
		func(ps core_api.Executor_exec_Params) error {
			if err := ps.SetBytecode(bc); err != nil {
				return err
			}

			ps.SetPpid(ppid)
			return ps.SetSession(core_api.Session(sess))
		})
	return Proc(f.Process()), release
}

// ExecFromCache behaves the same way as Exec, but expects the bytecode to be already
// cached at the executor.
func (ex Executor) ExecFromCache(ctx context.Context, sess core_api.Session, cid cid.Cid, ppid uint32) (Proc, capnp.ReleaseFunc) {
	f, release := core_api.Executor(ex).ExecCached(ctx,
		func(ps core_api.Executor_execCached_Params) error {
			if err := ps.SetCid(cid.Bytes()); err != nil {
				return err
			}

			ps.SetPpid(ppid)
			return ps.SetSession(core_api.Session(sess))
		})
	return Proc(f.Process()), release
}
