// package server exports the Wetware worker node.
package server

import (
	"context"

	"github.com/lthibault/log"
	ps "github.com/mikelsr/go-libp2p-pubsub"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"go.uber.org/fx"

	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/pkg/anchor"
	csp "github.com/wetware/ww/pkg/csp/server"
	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/pubsub"
	service "github.com/wetware/ww/pkg/registry"
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
	host.PubSubProvider
}

func (n Node) Loggable() map[string]any {
	return n.Vat.Loggable()
}

// Joiner is a factory type that builds a Node from configuration,
// and joins the cluster. Joiners SHOULD NOT be reused, and should
// be promptly discarded after a call to Join.
type Joiner struct {
	fx.In

	Cluster ClusterConfig `optional:"true"`
	// Runtime RuntimeConfig `optional:"true"`
}

// Join the cluster.  Note that callers MUST call Bootstrap() on
// the returned *Node to complete the bootstrap csp.
func (j Joiner) Join(vat casm.Vat, r Router) (*Node, error) {
	c, err := j.Cluster.New(vat, r)
	if err != nil {
		return nil, err
	}

	ps := j.pubsub(vat.Logger, r)
	vat.Export(host.Capability, host.Server{
		ViewProvider:     c,
		PubSubProvider:   ps,
		AnchorProvider:   j.anchor(),
		RegistryProvider: j.service(),
		ExecutorProvider: j.executor(),
	})

	return &Node{
		Vat:            vat,
		Cluster:        c,
		PubSubProvider: ps,
	}, nil
}

func (j Joiner) pubsub(log log.Logger, router pubsub.TopicJoiner) *pubsub.Server {
	return &pubsub.Server{
		Log:         log,
		TopicJoiner: router,
	}
}

func (j Joiner) service() service.Server {
	return service.Server{}
}

// TODO(soon):  return a host anchor instead of a generic anchor.
func (j Joiner) anchor() *anchor.Node {
	return new(anchor.Node) // root node
}

func (j Joiner) executor() csp.Server {
	// TODO find an elegant way of creating custom executors
	ctx := context.TODO()
	cache := wazero.NewCompilationCache()
	runtimeCfg := wazero.
		NewRuntimeConfigCompiler().
		WithCompilationCache(cache)
	r := wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		panic(err)
	}

	return csp.Server{
		ProcCounter: csp.AtomicCounter{},
		Runtime:     r,
		BcRegistry:  csp.RegistryServer{},
	}
}
