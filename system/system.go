package system

import (
	"context"
	_ "embed"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	casm "github.com/wetware/casm/pkg"
	"go.uber.org/fx"
)

// Vat returns an asynchronous API to a network host.
func Vat(ns string, h host.Host) casm.Vat {
	return casm.Vat{
		NS:   ns,
		Host: h,
	}
}

func Host(privkey crypto.PrivKey) (host.Host, error) {
	return libp2p.New(
		libp2p.Identity(privkey),
		libp2p.NoTransports,
		libp2p.Transport(quic.NewTransport),
		// libp2p.Transport(webtransport.New),  // FIXME
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(websocket.New))
}

func Router(h host.Host) (*dual.DHT, error) {
	return dual.New(context.Background(), h)
	//dual.DHTOption(),
	//dual.WanDHTOption(),
	//dual.LanDHTOption(),
}

func WithRouting() fx.Option {
	return fx.Options(
		fx.Provide(Router),
		fx.Decorate(func(h host.Host, dht *dual.DHT) host.Host {
			return routedhost.Wrap(h, dht)
		}))
}

func WithDefaultServer() fx.Option {
	return fx.Options(
		fx.Provide(Host, Vat, DefaultROM),
		WithRouting())
}
