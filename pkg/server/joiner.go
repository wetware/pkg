package server

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"

	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/ww/pkg/cap/anchor"
	pscap "github.com/wetware/ww/pkg/cap/pubsub"
)

type PubSub interface {
	Join(string, ...pubsub.TopicOpt) (*pubsub.Topic, error)
	RegisterTopicValidator(string, interface{}, ...pubsub.ValidatorOpt) error
	UnregisterTopicValidator(string) error
}

type Joiner struct {
	ns   string
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

func (j Joiner) Join(ctx context.Context, h host.Host, ps PubSub) (*Node, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("uuid: %w", err)
	}

	c, err := cluster.New(ctx, ps, j.options(h, id)...)
	if err != nil {
		return nil, fmt.Errorf("join cluster: %w", err)
	}

	var cs = newCapSet(
		anchor.New(c),
		pscap.New(ps, pscap.WithLogger(j.log)))

	var n = &Node{
		id: id,
		h:  h,
		c:  cs,
	}

	cs.registerRPC(n.h, j.log.With(n))

	return n, c.Bootstrap(ctx)
}

func (j Joiner) options(h host.Host, u uuid.UUID) []cluster.Option {
	log := j.log.
		WithField("id", h.ID()).
		WithField("ns", j.ns).
		WithField("instance", u)

	return append([]cluster.Option{
		cluster.WithLogger(log),
		cluster.WithNamespace(j.ns),
	}, j.opts...)
}
