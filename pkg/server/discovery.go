package server

import (
	"context"
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	disc "github.com/libp2p/go-libp2p-discovery"
	"github.com/lthibault/log"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/casm/pkg/pex"
	"go.uber.org/fx"
)

type DiscoveryFactory interface {
	New(context.Context, host.Host, routing.ContentRouting) (discovery.Discovery, error)
}

type PexDiscovery struct {
	fx.In

	Logger log.Logger

	NS  string        `name:"ns"`
	TTL time.Duration `name:"ttl"`

	Cluster   discovery.Advertiser
	Boot      discovery.Discoverer
	Datastore ds.Batching
}

func (p PexDiscovery) New(ctx context.Context, h host.Host, r routing.ContentRouting) (discovery.Discovery, error) {
	px, err := pex.New(ctx, h,
		pex.WithLogger(p.Logger),
		pex.WithDiscovery(p),
		pex.WithDatastore(p.Datastore))

	// If the namespace matches the cluster pubsub topic,
	// fetch peers from PeX, which itself will fall back
	// on the bootstrap service 'p'.
	return boot.Cache{
		Match: exactly(p.NS),
		Cache: px,
		Else:  disc.NewRoutingDiscovery(r),
	}, err
}

func (p PexDiscovery) Advertise(ctx context.Context, ns string, opt ...discovery.Option) (ttl time.Duration, err error) {
	// This is the lowest-level (and often most expensive) form of
	// advertising.  Implementations will vary substantially in their
	// semantics.
	if ttl, err = p.Cluster.Advertise(ctx, ns, opt...); err == nil {
		p.Logger.Debugf("advertised %s", ns)
	}

	return
}

func (p PexDiscovery) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (peers <-chan peer.AddrInfo, err error) {
	// This is the lowest-level (and often most expensive) form
	// of peeer discovery.  It is wrapped by PeX and called only
	// when we fail to bootstrap from a persisted local view.
	if peers, err = p.Boot.FindPeers(ctx, ns, opt...); err != nil {
		p.Logger.Debugf("booting %s", ns)
	}

	return
}

func exactly(match string) func(string) bool {
	return func(s string) bool {
		return match == s
	}
}
