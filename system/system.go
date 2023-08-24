package system

import (
	"context"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero"
)

// Declare the host module from a set of exported functions.
var HostModule wazergo.HostModule[*Module] = functions{
	"__send": wazergo.F1((*Module).Send),
}

func Instantiate(ctx context.Context, r wazero.Runtime) (*Socket, error) {
	pipe := make(chan segment, 16)

	instance, err := wazergo.Instantiate[*Module](ctx, r, HostModule,
		options(pipe)...)
	if err != nil {
		return nil, err
	}

	return &Socket{
		instance: instance,
		context:  ctx,
		recv:     pipe,
	}, nil
}

type Option = wazergo.Option[*Module]

func options(pipe chan<- segment) []Option {
	return []Option{
		withPipe(pipe)}
}

func withPipe(pipe chan<- segment) Option {
	return wazergo.OptionFunc(func(mod *Module) {
		mod.send = pipe
	})
}

// The `functions` type impements `HostModule[*Module]`, providing the
// module name, map of exported functions, and the ability to create instances
// of the module type.
type functions wazergo.Functions[*Module]

func (f functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Module] {
	return (wazergo.Functions[*Module])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Module, error) {
	mod := &Module{}
	wazergo.Configure(mod, opts...)
	return mod, nil
}

// Module will be the Go type we use to maintain the state of our module
// instances.
type Module struct {
	send chan<- segment
}

func (mod Module) Close(context.Context) error {
	// FIXME:  this might double-close if there is a concurrent
	// call to Send() and the context has not yet expired.  Not
	// sure if this is a problem in practice.
	close(mod.send)
	return nil
}

func (mod Module) Send(ctx context.Context, seg segment) types.Error {
	select {
	case mod.send <- seg:
		return types.OK
	case <-ctx.Done():
		return types.Fail(ctx.Err())
	}
}
