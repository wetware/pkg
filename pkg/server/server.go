// package server exports the Wetware worker node.
package server

import (
	"context"

	ps "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/routing"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/pubsub"
)

type Router interface {
	Join(string, ...ps.TopicOpt) (*ps.Topic, error)
	RegisterTopicValidator(topic string, val interface{}, opts ...ps.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
}

type Node struct {
	Vat     casm.Vat
	cluster *cluster.Router
	pubsub  interface{ UnregisterTopicValidator(string) error }
}

func (n Node) ID() routing.ID {
	return n.cluster.ID()
}

func (n Node) String() string {
	return n.cluster.String()
}

func (n Node) Loggable() map[string]any {
	return n.cluster.Loggable()
}

func (n Node) Bootstrap(ctx context.Context) error {
	return n.cluster.Bootstrap(ctx)
}

func (n Node) Close() error {
	n.cluster.Stop()
	return n.pubsub.UnregisterTopicValidator(n.Vat.NS)
}

type Joiner struct {
	fx.In

	Vat casm.Vat

	Log      log.Logger    `optional:"true"`
	Router   RoutingConfig `optional:"true"`
	Debugger DebugConfig   `optional:"true"`
}

func (j Joiner) Join(r Router) (*Node, error) {
	c, err := j.Router.Bind(r, j.Vat.NS)
	if err != nil {
		return nil, err
	}

	j.Vat.Export(host.Capability, host.Server{
		ViewProvider:   c,
		PubSubProvider: j.pubsub(r),
		AnchorProvider: j.anchor(),
		DebugProvider:  j.Debugger.New(),
	})

	return &Node{
		Vat:     j.Vat,
		cluster: c,
		pubsub:  r,
	}, nil
}

func (j Joiner) pubsub(router pubsub.TopicJoiner) *pubsub.Router {
	return &pubsub.Router{
		Log:         j.Log,
		TopicJoiner: router,
	}
}

func (j Joiner) anchor() anchor.Server {
	return anchor.Root()
}
