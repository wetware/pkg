package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/cluster"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	"github.com/wetware/ww/pkg/vat"

	cluster_cap "github.com/wetware/ww/pkg/ocap/cluster"
	pubsub_cap "github.com/wetware/ww/pkg/ocap/pubsub"
)

type PubSub interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
	RegisterTopicValidator(string, interface{}, ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(string) error
}

type Joiner struct {
	log     log.Logger
	opts    []cluster.Option
	metrics *statsdutil.WwMetricsRecorder
}

func NewJoiner(opt ...Option) Joiner {
	var j Joiner
	for _, option := range withDefault(opt) {
		option(&j)
	}
	return j
}

func (j Joiner) Join(ctx context.Context, vat vat.Network, ps PubSub) (*Node, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// generate instance ID
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("uuid: %w", err)
	}

	// join the cluster topic
	c, err := cluster.New(ctx, ps, j.options(vat, id)...)
	if err != nil {
		return nil, fmt.Errorf("join cluster: %w", err)
	}

	// add metric provider
	j.metrics.Add(ClusterMetrics{View: c.View()})

	// export default capabilities
	logger := j.log.With(vat)

	vat.Export(
		pubsub_cap.Capability,
		pubsub_cap.New(vat.NS, ps, pubsub_cap.WithLogger(logger)))

	vat.Export(
		cluster_cap.HostCapability,
		&cluster_cap.HostServer{
			RoutingTable: c.View()},
	)

	// etc ...

	// Bootstrap the node
	return &Node{
		id:  id,
		vat: vat,
		c:   c,
	}, c.Bootstrap(ctx)
}

func (j Joiner) options(vat vat.Network, u uuid.UUID) []cluster.Option {
	log := j.log.
		WithField("id", vat.Host.ID()).
		WithField("ns", vat.NS).
		WithField("instance", u)

	return append([]cluster.Option{
		cluster.WithLogger(log),
		cluster.WithNamespace(vat.NS),
	}, j.opts...)
}

type ClusterMetrics struct {
	cluster.View
}

func (c ClusterMetrics) Metrics() map[string]interface{} {
	metrics := make(map[string]interface{}, 0)

	view_size := 0
	for it := c.Iter(); it.Record() != nil; it.Next() {
		view_size++
	}

	metrics["view.size"] = view_size
	return metrics
}
