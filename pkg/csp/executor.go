package csp

import (
	"context"

	capnp "capnproto.org/go/capnp/v3"

	api "github.com/wetware/ww/api/process"
)

// Executor is a capability that can spawn processes.
type Executor api.Executor

func (ex Executor) AddRef() Executor {
	return Executor(capnp.Client(ex).AddRef())
}

func (ex Executor) Release() {
	capnp.Client(ex).Release()
}

func (ex Executor) Exec(ctx context.Context, src []byte, ppid uint32, caps ...capnp.Client) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).Exec(ctx, func(ps api.Executor_exec_Params) error {
		if err := ps.SetBytecode(src); err != nil {
			return err
		}
		if caps == nil {
			return nil
		}
		cl, err := ClientsToNewList(caps...)
		if err != nil {
			return err
		}
		ps.SetPpid(ppid)
		return ps.SetCaps(cl)
	})
	return Proc(f.Process()), release
}

func (ex Executor) ExecFromCache(ctx context.Context, hash []byte, ppid uint32, caps ...capnp.Client) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).ExecFromCache(ctx, func(ps api.Executor_execFromCache_Params) error {
		if err := ps.SetHash(hash); err != nil {
			return err
		}
		if caps == nil {
			return nil
		}
		cl, err := ClientsToNewList(caps...)
		if err != nil {
			return err
		}

		ps.SetPpid(ppid)
		return ps.SetCaps(cl)
	})
	return Proc(f.Process()), release
}
