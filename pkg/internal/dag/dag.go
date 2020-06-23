package dag

import (
	"context"

	"go.uber.org/fx"

	"github.com/ipfs/go-blockservice"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipfs/go-merkledag"
)

// Config for DAG service.
type Config struct {
	fx.In

	Blocks blockservice.BlockService
}

// Module containing DAG primitives.
type Module struct {
	fx.Out

	DAG format.DAGService
}

// New DAG module
func New(ctx context.Context, cfg Config) (mod Module, err error) {
	// TODO(deprecate):  merkeldag is slated for deprecation (horizon unknown)
	mod.DAG = merkledag.NewDAGService(cfg.Blocks)
	return
}
