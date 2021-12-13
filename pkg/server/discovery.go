package server

import (
	"time"

	ds "github.com/ipfs/go-datastore"
	"github.com/jbenet/goprocess"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	disc "github.com/libp2p/go-libp2p-discovery"
	"github.com/lthibault/log"
	ctxutil "github.com/lthibault/util/ctx"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/pex"
	"github.com/wetware/ww/pkg/boot"
	"go.uber.org/fx"
)

type DiscoveryFactory interface {
	New(host.Host, BootStrategy, routing.ContentRouting) (discovery.Discovery, error)
}

type PexDiscovery struct {
	fx.In

	Logger log.Logger

	NS        string        `name:"ns"`
	TTL       time.Duration `name:"ttl"`
	Datastore ds.Batching
}

func (p PexDiscovery) New(h host.Host, b BootStrategy, r routing.ContentRouting) (discovery.Discovery, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())

	bootstrap, err := b.New(h)
	if err != nil {
		return nil, err
	}

	// Wrap the bootstrap discovery service in a peer sampling service.
	px, err := pex.New(ctx, h,
		pex.WithLogger(p.Logger),
		pex.WithDiscovery(bootstrap),
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

type BootStrategy interface {
	New(h host.Host) (discovery.Discovery, error)
}

type PortScanStrategy struct {
	boot.PortListener
	boot.PortKnocker
}

func (p *PortScanStrategy) New(h host.Host) (discovery.Discovery, error) {
	bus := h.EventBus()

	// We update the local peer record each time the host binds
	// or unbinds a network address.
	sub, err := bus.Subscribe(new(event.EvtLocalAddressesUpdated))
	if err != nil {
		return nil, err
	}
	goprocess.WithParent(h.Network().Process()).SetTeardown(sub.Close)

	// Initialize the register.  The subscription is stateful, so this
	// will not block.
	v := <-sub.Out()
	ev := v.(event.EvtLocalAddressesUpdated)
	p.PortListener.Endpoint.Register = casm.New(ev.SignedPeerRecord)
	p.PortKnocker.RequestBody = ev.SignedPeerRecord

	// Update the register in the background in case the host (un)binds
	// any addresses.
	go func() {
		for v := range sub.Out() {
			ev := v.(event.EvtLocalAddressesUpdated)
			p.PortListener.Store(ev.SignedPeerRecord)
		}
	}()

	return p, nil
}

func exactly(match string) func(string) bool {
	return func(s string) bool {
		return match == s
	}
}
