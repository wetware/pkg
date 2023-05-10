package runtime

import (
	"github.com/tetratelabs/wazero"
	"go.uber.org/fx"
)

func (c Config) WASM() fx.Option {
	if c.wasmConfig == nil {
		return fx.Options()
	}

	rc := fx.Annotate(c.wasmConfig, fx.As(new(wazero.RuntimeConfig)))
	return fx.Supply(rc)
}
