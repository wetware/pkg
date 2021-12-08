package server

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	ctxutil "github.com/lthibault/util/ctx"
)

type DHT interface {
	routing.Routing
	routing.PubKeyFetcher
}

type DHTFactory interface {
	New(host.Host) (DHT, error)
}

type RoutingHook interface {
	SetRouting(DHTFactory)
}

type DualDHTFactory []dual.Option

func (opt DualDHTFactory) New(h host.Host) (DHT, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())

	if len(opt) == 0 {
		opt = append(opt,
			dual.LanDHTOption(dht.Mode(dht.ModeServer)),
			dual.WanDHTOption(dht.Mode(dht.ModeAuto)))
	}

	return dual.New(ctx, h, opt...)
}

type routingHook struct {
	DHTFactory
	dht DHT
	err error
}

func (r *routingHook) New(h host.Host) (DHT, error) {
	if r.dht == nil && r.err == nil {
		r.dht, r.err = r.DHTFactory.New(h)
	}

	return r.dht, r.err
}

func (r *routingHook) Option() libp2p.Option {
	return libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		return r.New(h)
	})
}
