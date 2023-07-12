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

// TODO mikel clean and move to appropiate file
func clientsToList(caps ...capnp.Client) (capnp.PointerList, error) {
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return capnp.PointerList{}, err
	}
	l, err := capnp.NewPointerList(seg, int32(len(caps)))
	if err != nil {
		return capnp.PointerList{}, err
	}
	for i, cap := range caps {
		_, iSeg, err := capnp.NewMessage(capnp.SingleSegment(nil))
		if err != nil {
			return capnp.PointerList{}, err
		}
		l.Set(i, cap.EncodeAsPtr(iSeg))
	}
	return l, nil
}

func (ex Executor) Exec(ctx context.Context, src []byte, caps ...capnp.Client) (Proc, capnp.ReleaseFunc) {
	f, release := api.Executor(ex).Exec(ctx, func(ps api.Executor_exec_Params) error {
		if err := ps.SetBytecode(src); err != nil {
			return err
		}
		if caps == nil {
			return nil
		}
		c, err := clientsToList(caps...)
		if err != nil {
			return err
		}
		return ps.SetCaps(c)
	})
	return Proc(f.Process()), release
}
