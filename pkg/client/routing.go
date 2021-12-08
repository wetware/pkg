package client

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	ctxutil "github.com/lthibault/util/ctx"
)

type RoutingFactory interface {
	New(host.Host) (routing.Routing, error)
}

type RoutingHook interface {
	SetRouting(RoutingFactory)
}

type defaultRoutingFactory struct{}

func (defaultRoutingFactory) New(h host.Host) (routing.Routing, error) {
	ctx := ctxutil.C(h.Network().Process().Closing())
	return dual.New(ctx, h, dual.DHTOption(dht.Mode(dht.ModeClient)))
}

type routingHook struct {
	RoutingFactory
	r   routing.Routing
	err error
}

func (r *routingHook) New(h host.Host) (routing.Routing, error) {
	if r.r == nil && r.err == nil {
		r.r, r.err = r.RoutingFactory.New(h)
	}

	return r.r, r.err
}

func (r *routingHook) Option() libp2p.Option {
	return libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		return r.New(h)
	})
}
