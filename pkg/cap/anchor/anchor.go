//go:generate mockgen -destination ../../../internal/test/mock/pkg/cap/anchor/mock_anchor.go github.com/wetware/ww/pkg/cap/anchor Cluster

package anchor

import (
	"github.com/wetware/casm/pkg/cluster"
)

type Cluster interface {
	Close() error
	String() string
	View() cluster.View
}

type Factory struct {
	c Cluster
}

func New(c Cluster) Factory {
	return Factory{
		c: c,
	}
}

// TODO - uncomment

// func (f Factory) New(p *server.Policy) Anchor {
// 	if p == nil {
// 		p = &defaultPolicy
// 	}

// 	return Root(api.Anchor_ServerToClient(f, p))
// }

// String returns the cluster namespace
func (f Factory) String() string {
	return f.c.String()
}

func (f Factory) Close() (err error) {
	if f.c != nil {
		err = f.c.Close()
	}

	return
}
