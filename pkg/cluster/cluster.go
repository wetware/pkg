//go:generate mockgen -destination ../../internal/test/mock/pkg/cluster/mock_cluster.go github.com/wetware/ww/pkg/cluster Announcer,Clock,PeerSet

// Package cluster contains primitives for Wetware's pubsub-based clustering protocol.
package cluster

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/fx"
)

// Announcer can notify a cluster of the bound host's presence on the network.
type Announcer interface {
	Namespace() string
	Announce(ctx context.Context, ttl time.Duration) error
}

// Clock tracks the host's local epoch.
// If the current epoch exceeds a remote peer's TTL,
// the peer will be removed from the peerset.
type Clock interface {
	// Advance the timer to the current epoch.  Implementations SHOULD panic if the
	// current epoch <= the previous epoch.
	Advance(epoch time.Time)
}

// PeerSet tracks the liveliness of peers in the cluster.
type PeerSet interface {
	Peers() peer.IDSlice
	Contains(peer.ID) bool
	Upsert(peer.ID, uint64, time.Duration) bool
}

// Config for cluster.
type Config struct {
	fx.In

	Namespace string `name:"ns"`
	PubSub    *pubsub.PubSub
}

// Module for cluster.
type Module struct {
	fx.Out

	Announcer Announcer
	Clock     Clock
	Cluster   PeerSet
}

// New cluster module.  The resulting module provides the caller with the full set of
// interfaces necessary participate in the Wetware cluster protocol.  This is typically
// used by Hosts.
func New(ctx context.Context, cfg Config, lx fx.Lifecycle) (Module, error) {
	var f filter

	validator := newHeartbeatValidator(&f)
	if err := cfg.PubSub.RegisterTopicValidator(cfg.Namespace, validator); err != nil {
		return Module{}, err
	}

	t, err := cfg.PubSub.Join(cfg.Namespace)
	if err != nil {
		return Module{}, err
	}

	lx.Append(consumer(ctx, t))

	return Module{
		Announcer: announcer{t},
		Clock:     &f,
		Cluster:   &f,
	}, nil
}
