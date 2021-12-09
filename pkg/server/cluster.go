package server

import (
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/cluster"
	"github.com/wetware/casm/pkg/cluster/pulse"
	"go.uber.org/fx"
)

type ClusterConfig struct {
	fx.In

	NS      string               `optional:"true" name:"ns"`
	Log     log.Logger           `optional:"true"`
	TTL     time.Duration        `optional:"true" name:"ttl"`
	Meta    pulse.Preparer       `optional:"true"`
	Routing cluster.RoutingTable `optional:"true"`
	Ready   pubsub.RouterReady   `optional:"true"`
}

func (cc ClusterConfig) options() []cluster.Option {
	return []cluster.Option{
		cluster.WithNamespace(cc.NS),
		cluster.WithLogger(cc.Log),
		cluster.WithTTL(cc.TTL),
		cluster.WithRoutingTable(cc.Routing),
		cluster.WithReadiness(cc.Ready),
		cluster.WithMeta(cc.Meta)}
}
