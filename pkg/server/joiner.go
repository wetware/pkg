package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"golang.org/x/sync/errgroup"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/pkg/vat"

	// exported capabilities
	anchor_cap "github.com/wetware/ww/pkg/cap/anchor"
	cluster_cap "github.com/wetware/ww/pkg/cap/cluster"
	pubsub_cap "github.com/wetware/ww/pkg/cap/pubsub"
)

type PubSub interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
	RegisterTopicValidator(string, interface{}, ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(string) error
}

type Joiner struct {
	log      log.Logger
	newMerge func(vat.Network) anchor_cap.MergeStrategy
	opts     []cluster.Option
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

	// export default capabilities
	vat.Export(
		pubsub_cap.Capability,
		pubsub_cap.New(vat.NS, ps, pubsub_cap.WithLogger(j.log.With(vat))))

	vat.Export(
		cluster_cap.ViewCapability,
		cluster_cap.ViewServer{RoutingTable: c.View()})

	vat.Export(
		anchor_cap.AnchorCapability,
		anchor_cap.NewHost(j.newMerge(vat)))

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

type basicMerge struct{ host.Host }

func newMergeFactory(m anchor_cap.MergeStrategy) func(vat.Network) anchor_cap.MergeStrategy {
	if m != nil {
		return func(vat.Network) anchor_cap.MergeStrategy { return m }
	}

	return func(vat vat.Network) anchor_cap.MergeStrategy {
		return basicMerge{vat.Host}
	}
}

func (m basicMerge) Merge(ctx context.Context, peers []peer.AddrInfo) error {
	var g errgroup.Group

	for _, info := range peers {
		g.Go(m.connector(ctx, info))
	}

	return g.Wait()
}

func (m basicMerge) connector(ctx context.Context, info peer.AddrInfo) func() error {
	return func() (err error) {
		if err = m.Connect(ctx, info); err != nil {
			err = fmt.Errorf("%s: %w", info.ID.ShortString(), err)
		}

		return
	}
}
