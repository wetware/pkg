package server

import (
	"context"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/wetware/pkg/util/proto"
)

func (conf Config) withRouting() (*routedhost.RoutedHost, *dual.DHT, error) {
	dht, err := conf.newDHT(context.TODO(), conf.Host)
	if err != nil {
		return nil, nil, err
	}

	return routedhost.Wrap(conf.Host, dht), dht, nil
}

func (conf Config) newDHT(ctx context.Context, h host.Host) (*dual.DHT, error) {
	// TODO:  Use dht.BootstrapPeersFunc to get bootstrap peers from PeX?
	//        This might allow us to greatly simplify our architecture and
	//        runtime initialization.  In particular:
	//
	//          1. The DHT could query PeX directly, eliminating the need for
	//             dynamic dispatch via boot.Namespace.
	//
	//          2. The server.Joiner type could be simplified, and perhaps
	//             eliminated entirely.

	return dual.New(ctx, h,
		dual.LanDHTOption(lanOpt(conf.NS)...),
		dual.WanDHTOption(wanOpt(conf.NS)...))
}

func lanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("lan")}
}

func wanOpt(ns string) []dht.Option {
	return []dht.Option{
		dht.Mode(dht.ModeAuto),
		dht.ProtocolPrefix(proto.Root(ns)),
		dht.ProtocolExtension("wan")}
}
