package start

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-kad-dht/dual"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	ww "github.com/wetware/ww/pkg"
)

var defaultHostOpt = []libp2p.Option{
	libp2p.NoTransports,
	libp2p.Transport(libp2pquic.NewTransport),
	libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/2020/quic", "/ip6/::/udp/2020/quic"),
}

type hostConfig struct {
	fx.In

	CLI     *cli.Context
	HostOpt []libp2p.Option `optional:"true"`
}

func (cfg hostConfig) hostOpt() []libp2p.Option {
	if len(cfg.HostOpt) == 0 {
		cfg.HostOpt = defaultHostOpt
	}

	if cfg.CLI.IsSet("secret") {
		cfg.HostOpt = append(cfg.HostOpt,
			libp2p.PrivateNetwork([]byte(cfg.CLI.String("secret"))))
	}

	return cfg.HostOpt
}

func newRoutedHost(ctx context.Context, cfg hostConfig, lx fx.Lifecycle) (host.Host, ww.DHT, error) {
	h, err := libp2p.New(ctx, cfg.hostOpt()...)
	if err != nil {
		return nil, nil, err
	}

	d, err := dual.New(ctx, h)
	if err == nil {
		lx.Append(closer(d))

		h = routedhost.Wrap(h, d)
		lx.Append(closer(h))
	}

	return h, d, err
}
