// package server exports the Wetware worker node.
package server

import (
	"context"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
)

type Node struct {
	Vat casm.Vat
	*cluster.Node
}

func (n Node) Bootstrap(ctx context.Context) error {
	return n.Node.Bootstrap(ctx)
}
