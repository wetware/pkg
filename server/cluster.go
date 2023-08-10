package server

import (
	"context"
	"os"
	"time"

	"github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/wetware/pkg/cluster"
	"github.com/wetware/pkg/cluster/pulse"
	"github.com/wetware/pkg/cluster/routing"
)

type clusterConfig struct {
	Host      host.Host
	PubSub    *pubsub.PubSub
	Discovery discovery.Discovery
	DHT       *dual.DHT
}

func (cfg Config) newCluster(ctx context.Context, cc clusterConfig) (*cluster.Router, error) {
	rt := routing.New(time.Now())

	err := cc.PubSub.RegisterTopicValidator(cfg.NS, pulse.NewValidator(rt))
	if err != nil {
		return nil, err
	}

	t, err := cc.PubSub.Join(cfg.NS)
	if err != nil {
		return nil, err
	}

	return &cluster.Router{
		Topic:        t,
		Log:          cfg.Logger,
		Meta:         cfg.preparer(),
		RoutingTable: rt,
	}, nil
}

type metaFieldSlice []routing.MetaField

func (cfg Config) preparer() pulse.Preparer {
	fs := make(metaFieldSlice, 0, len(cfg.Meta))

	for key, value := range cfg.Meta {
		fs = append(fs, routing.MetaField{
			Key:   key,
			Value: value,
		})
	}

	return fs
}

func (m metaFieldSlice) Prepare(h pulse.Heartbeat) error {
	if err := h.SetMeta(m); err != nil {
		return err
	}

	// hostname may change over time
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	return h.SetHost(host)
}
