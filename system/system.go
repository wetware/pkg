package system

import (
	"context"
	_ "embed"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/lthibault/log"
	casm "github.com/wetware/casm/pkg"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

// Vat returns an asynchronous API to a network host.
func Vat(log log.Logger, ns string, h host.Host) casm.Vat {
	return casm.Vat{
		NS:   ns,
		Host: h,
		Logger: log.
			WithField("ns", ns).
			WithField("peer", h.ID()),
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

func Router(lx fx.Lifecycle, ns string, h host.Host) (*dual.DHT, error) {
	dht, err := dual.New(context.Background(), h,
		lanOpt(ns),
		wanOpt(ns))
	if err == nil {
		lx.Append(fx.StopHook(dht.Close))
	}

	return dht, err
}

func lanOpt(ns string) dual.Option {
	return dual.LanDHTOption(
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ww.Subprotocol(ns)),
		dht.ProtocolExtension("lan"))
}

func wanOpt(ns string) dual.Option {
	return dual.WanDHTOption(
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(ww.Subprotocol(ns)),
		dht.ProtocolExtension("wan"))
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
		fx.Provide(ListenBoot, Host, Vat, PubSub, DefaultROM),
		WithRouting())
}
