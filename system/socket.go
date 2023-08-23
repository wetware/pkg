package system

import (
	"context"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
)

type Socket struct{}

func (sock *Socket) Close() error {
	return nil
}

func (sock *Socket) Sched(ctx context.Context) func() {
	// TODO:  this needs to pass input into the guest
	return runtime.Gosched
}

func (sock *Socket) Transport() rpc.Transport {
	panic("NOT IMPLEMENTED")
}
