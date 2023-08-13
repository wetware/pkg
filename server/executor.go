package server

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	csp_server "github.com/wetware/pkg/cap/csp/server"
)

type executorConfig struct {
	Cache      csp_server.BytecodeCache
	RuntimeCfg wazero.RuntimeConfig
}

func (cfg Config) newExecutor(ctx context.Context, ec executorConfig) (csp_server.Runtime, error) {
	r := wazero.NewRuntimeWithConfig(ctx, ec.RuntimeCfg)
	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		return csp_server.Runtime{}, err
	}

	return csp_server.Runtime{
		Runtime: r,
		Cache:   ec.Cache,
	}, nil
}
