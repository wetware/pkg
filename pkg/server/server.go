// package server exports the Wetware worker node.
package server

import (
	"context"

	ps "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/process"
	"github.com/wetware/ww/pkg/pubsub"
)

// Router provides an interface for routing messages by topic, and supports
// per-message validation.   It is used by the Joiner to create the cluster
// topic through which heartbeat messages are routed.
type Router interface {
	Join(string, ...ps.TopicOpt) (*ps.Topic, error)
	RegisterTopicValidator(topic string, val interface{}, opts ...ps.ValidatorOpt) error
	UnregisterTopicValidator(topic string) error
}

// Cluster is a local model of the global Wetware cluster.  It models the
// cluster as a PA/EL system and makes no consistency guarantees.
//
// https://en.wikipedia.org/wiki/PACELC_theorem
type Cluster interface {
	Bootstrap(context.Context) error
	View() cluster.View
	String() string
	Close() error
}

// Node is a peer in the Wetware cluster.  Manually populating Node's fields
// is NOT RECOMMENDED.  Use Joiner instead.
type Node struct {
	Vat casm.Vat
	Cluster
}

func (n Node) Loggable() map[string]any {
	return n.Vat.Loggable()
}

// Joiner is a factory type that builds a Node from configuration,
// and joins the cluster. Joiners SHOULD NOT be reused, and should
// be promptly discarded after a call to Join.
type Joiner struct {
	fx.In

	Cluster  ClusterConfig `optional:"true"`
	Runtime  RuntimeConfig `optional:"true"`
	Debugger DebugConfig   `optional:"true"`
}

// Join the cluster.  Note that callers MUST call Bootstrap() on
// the returned *Node to complete the bootstrap process.
func (j Joiner) Join(vat casm.Vat, r Router) (*Node, error) {
	c, err := j.Cluster.New(vat, r)
	if err != nil {
		return nil, err
	}

	vat.Export(host.Capability, host.Server{
		ViewProvider:     c,
		PubSubProvider:   j.pubsub(vat.Logger, r),
		AnchorProvider:   j.anchor(),
		DebugProvider:    j.Debugger.New(),
		ExecutorProvider: j.executor(),
	})

	return &Node{
		Vat:     vat,
		Cluster: c,
	}, nil
}

func (j Joiner) pubsub(log log.Logger, router pubsub.TopicJoiner) *pubsub.Server {
	return &pubsub.Server{
		Log:         log,
		TopicJoiner: router,
	}
}

func (j Joiner) anchor() anchor.Server {
	return anchor.Root()
}

func (j Joiner) executor() process.Server {
	return process.Server{
		Runtime: j.Runtime.New(),
	}
}
