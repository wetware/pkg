// Package proc provides the WebAssembly host module for Wetware processes
package proc

import (
	"context"

	"github.com/stealthrocket/wazergo"
	"github.com/stealthrocket/wazergo/types"
	wasm "github.com/tetratelabs/wazero"
)

var fs wazergo.HostModule[*Module] = functions{
	"__test": wazergo.F2((*Module).Add),
}

// BindModule instantiates a host module instance and binds it to the supplied
// runtime.  The returned module instance is bound to the lifetime of r.
func BindModule(ctx context.Context, r wasm.Runtime) *wazergo.ModuleInstance[*Module] {
	// We use BindModule to avoid exporting fs as a public variable, since this
	// would allow users to mutate it.
	return wazergo.MustInstantiate(ctx, r, fs)
}

type functions wazergo.Functions[*Module]

func (functions) Name() string {
	return "ww"
}

func (f functions) Functions() wazergo.Functions[*Module] {
	return (wazergo.Functions[*Module])(f)
}

func (f functions) Instantiate(ctx context.Context, opts ...Option) (*Module, error) {
	module := &Module{}
	wazergo.Configure(module, opts...)
	return module, nil
}

type Option = wazergo.Option[*Module]

type Module struct{}

func (m Module) Close(context.Context) error {
	return nil
}

func (m Module) Add(ctx context.Context, a, b types.Uint32) types.Uint32 {
	return a + b
}
