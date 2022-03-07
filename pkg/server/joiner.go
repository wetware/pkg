package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/cluster"
	clcap "github.com/wetware/ww/pkg/cap/cluster"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
	"github.com/wetware/ww/pkg/vat"
)

type PubSub interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
	RegisterTopicValidator(string, interface{}, ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(string) error
}

type Joiner struct {
	log  log.Logger
	opts []cluster.Option
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
		pscap.Capability,
		pscap.New(vat.NS, ps, pscap.WithLogger(j.log.With(vat))))

	vat.Export(
		clcap.ViewCapability,
		clcap.NewViewServer(c.View()))

	vat.Export(
		clcap.AnchorCapability,
		clcap.NewHostAnchorServer(vat))

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
