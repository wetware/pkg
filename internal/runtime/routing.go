package runtime

import (
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	casm "github.com/wetware/casm/pkg"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

func (c Config) Routing() fx.Option {
	return fx.Options(
		fx.Provide(c.newDHT),
		fx.Decorate(routedHost))
}

func (c Config) newDHT(env Env, lx fx.Lifecycle, vat casm.Vat) (*dual.DHT, error) {
	// TODO:  Use dht.BootstrapPeersFunc to get bootstrap peers from PeX?
	//        This might allow us to greatly simplify our architecture and
	//        runtime initialization.  In particular:
	//
	//          1. The DHT could query PeX directly, eliminating the need for
	//             dynamic dispatch via boot.Namespace.
	//
	//          2. The server.Joiner type could be simplified, and perhaps
	//             eliminated entirely.

	d, err := dual.New(env.Context(), vat.Host,
		dual.LanDHTOption(lanOpt(env)...), // TODO:  options (w/ defaults) from Config
		dual.WanDHTOption(wanOpt(env)...)) // TODO:  options (w/ defaults) from Config

	if err == nil {
		lx.Append(fx.Hook{
			OnStart: d.Bootstrap,
			OnStop:  onclose(d),
		})
	}

	return d, err
}

func routedHost(h host.Host, dht *dual.DHT) host.Host {
	return routedhost.Wrap(h, dht)
}

func lanOpt(env Env) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ww.Subprotocol(env.String("ns"))),
		dht.ProtocolExtension("lan")}
}

func wanOpt(env Env) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(ww.Subprotocol(env.String("ns"))),
		dht.ProtocolExtension("wan")}
}
