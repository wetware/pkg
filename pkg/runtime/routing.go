package runtime

import (
	"context"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	casm "github.com/wetware/casm/pkg"
	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

func (c Config) Routing() fx.Option {
	return fx.Options(
		fx.Provide(c.newDHT),
		fx.Decorate(routedVat))
}

type routingConfig struct {
	fx.In

	Ctx  context.Context
	Flag Flags
	Vat  casm.Vat
}

func (c Config) newDHT(config routingConfig, lx fx.Lifecycle) (*dual.DHT, error) {
	// TODO:  Use dht.BootstrapPeersFunc to get bootstrap peers from PeX?
	//        This might allow us to greatly simplify our architecture and
	//        runtime initialization.  In particular:
	//
	//          1. The DHT could query PeX directly, eliminating the need for
	//             dynamic dispatch via boot.Namespace.
	//
	//          2. The server.Joiner type could be simplified, and perhaps
	//             eliminated entirely.

	d, err := dual.New(config.Ctx, config.Vat.Host,
		dual.LanDHTOption(lanOpt(config.Flag)...), // TODO:  options (w/ defaults) from Config
		dual.WanDHTOption(wanOpt(config.Flag)...)) // TODO:  options (w/ defaults) from Config

	if err == nil {
		lx.Append(fx.Hook{
			OnStart: d.Bootstrap,
			OnStop:  onclose(d),
		})
	}

	return d, err
}

// NOTE:  we cannot pass host.Host as an argument, as it isn't registered with Fx.
func routedVat(vat casm.Vat, dht *dual.DHT) casm.Vat {
	vat.Host = routedhost.Wrap(vat.Host, dht)
	return vat
}

func lanOpt(flag Flags) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ww.Subprotocol(flag.String("ns"))),
		dht.ProtocolExtension("lan")}
}

func wanOpt(flag Flags) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(ww.Subprotocol(flag.String("ns"))),
		dht.ProtocolExtension("wan")}
}
