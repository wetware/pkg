package csp

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/ww/api/process"
)

type Args api.Args

// NewArgs creates a new Args capability from a list of strings.
func NewArgs(args ...string) Args {
	a := argsServer{
		args: args,
	}
	return Args(api.Args_ServerToClient(a))
}

// Args reads the args from an Args capability.
func (a Args) Args(ctx context.Context) ([]string, error) {
	f, _ := api.Args(a).Args(ctx, nil)
	<-f.Done()

	res, err := f.Struct()
	if err != nil {
		return nil, err
	}

	tl, err := res.Args()
	if err != nil {
		return nil, err
	}

	results := make([]string, tl.Len())
	for i := 0; i < tl.Len(); i++ {
		results[i], err = tl.At(i)
		if err != nil {
			break
		}
	}
	return results, err
}

// argsServer implements api.Args.
// It can be left in this package because it can be compiled to WASM.
type argsServer struct {
	args []string
}

func (a argsServer) Args(ctx context.Context, call api.Args_args) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return err
	}

	tl, err := capnp.NewTextList(seg, int32(len(a.args)))
	if err != nil {
		return nil
	}

	for i, arg := range a.args {
		tl.Set(i, arg)
	}

	return res.SetArgs(tl)
}
