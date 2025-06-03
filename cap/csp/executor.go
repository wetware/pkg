package csp

import (
	"context"
	"encoding/hex"

	capnp "capnproto.org/go/capnp/v3"
	"lukechampine.com/blake3"

	"github.com/ipfs/go-cid"
	core_api "github.com/wetware/pkg/api/core"
	proc_api "github.com/wetware/pkg/api/process"
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
func (ex Executor) Exec(
	ctx context.Context,
	sess core_api.Session,
	bc []byte,
	ppid uint32,
	argv ...string,
) (Proc, capnp.ReleaseFunc) {
	f, release := core_api.Executor(ex).Exec(ctx,
		func(ps core_api.Executor_exec_Params) error {
			args, err := EncodeTextList(argv)
			if err != nil {
				return err
			}
			ps.SetArgs(args)

			if err = ps.SetBytecode(bc); err != nil {
				return err
			}

			ps.SetPpid(ppid)
			return ps.SetSession(core_api.Session(sess))
		})
	return Proc(f.Process()), release
}

// ExecCached behaves the same way as Exec, but expects the bytecode to be already
// cached at the executor.
func (ex Executor) ExecCached(
	ctx context.Context,
	sess core_api.Session,
	cid cid.Cid,
	ppid uint32,
	argv ...string,
) (Proc, capnp.ReleaseFunc) {
	f, release := core_api.Executor(ex).ExecCached(ctx,
		func(ps core_api.Executor_execCached_Params) error {
			args, err := EncodeTextList(argv)
			if err != nil {
				return err
			}
			ps.SetArgs(args)

			if err = ps.SetCid(cid.Bytes()); err != nil {
				return err
			}

			ps.SetPpid(ppid)
			return ps.SetSession(core_api.Session(sess))
		})
	return Proc(f.Process()), release
}

// Get information about every running process in an executor.
func (ex Executor) Ps(ctx context.Context) ([]proc_api.Info, capnp.ReleaseFunc, error) {
	f, release := core_api.Executor(ex).Ps(ctx, nil)
	<-f.Done()
	s, err := f.Struct()
	if err != nil {
		return nil, release, err
	}

	pl, err := s.Procs()
	if err != nil {
		return nil, release, err
	}
	procs := make([]proc_api.Info, pl.Len())
	for i := 0; i < pl.Len(); i++ {
		procs[i] = pl.At(i)
	}

	return procs, release, nil
}

// DecodeTextList creates a string slice from a capnp.TextList.
func DecodeTextList(l capnp.TextList) ([]string, error) {
	var err error
	v := make([]string, l.Len())
	for i := 0; i < l.Len(); i++ {
		v[i], err = l.At(i)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}

// EncodeTextList creates a capnp.TextList from a string slice.
func EncodeTextList(v []string) (capnp.TextList, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	l, err := capnp.NewTextList(seg, int32(len(v)))
	if err != nil {
		return capnp.TextList{}, err
	}

	for i := 0; i < len(v); i++ {
		if err = l.Set(i, v[i]); err != nil {
			return capnp.TextList{}, err
		}
	}
	return l, nil
}

func (ex Executor) DialPeer(ctx context.Context, peer []byte) (core_api.Session, bool, error) {
	b := make([]byte, len(peer))
	copy(b, peer)
	f, _ := core_api.Executor(ex).DialPeer(ctx, func(e core_api.Executor_dialPeer_Params) error {
		return e.SetPeerId(b)
	})
	// defer release()
	<-f.Done()
	s, err := f.Struct()
	if err != nil {
		panic(err)
	}
	if s.Self() {
		return core_api.Session{}, true, nil
	}
	sess, err := s.Session()
	return sess, false, err
}
