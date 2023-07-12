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

func (ex Executor) Exec(ctx context.Context, src []byte, caps ...capnp.Client) (Proc, capnp.ReleaseFunc) {
	/**/
	// TODO mikel is this requeted?
	// for i, c := range caps {
	// 	if err := c.Resolve(ctx); err != nil {
	// 		panic(err)
	// 	}
	// 	caps[i] = c.AddRef()
	// }
	/**/
	f, release := api.Executor(ex).Exec(ctx, func(ps api.Executor_exec_Params) error {
		// FIXME caps wont resolve here!
		// check out https://github.com/capnproto/go-capnp/issues/244
		if err := ps.SetBytecode(src); err != nil {
			return err
		}
		if caps == nil {
			return nil
		}
		// TODO mikel I might have over-engineered this
		cl, err := ps.NewCaps(int32(len(caps)))
		if err != nil {
			return err
		}
		if err = ClientsToExistingList(&cl, caps...); err != nil {
			return err
		}

		return ps.SetCaps(cl)
	})
	return Proc(f.Process()), release
}
