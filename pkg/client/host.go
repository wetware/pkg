package client

import (
	"context"
	"time"

	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"

	p2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/pnet"
	"github.com/pkg/errors"
)

// we need to return a host.Host before it's actually initialized.
type hostWrapper struct{ host.Host }

func newBaseContext(lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

type hostParams struct {
	fx.In

	Ctx      context.Context
	Discover Discover
	PSK      pnet.PSK
}

func newHost(lx fx.Lifecycle, p hostParams) *hostWrapper {
	var hw hostWrapper

	hookHost(lx, &hw, p)
	hookDiscover(lx, &hw, p.Discover)

	return &hw
}

func hookHost(lx fx.Lifecycle, hw *hostWrapper, p hostParams) {
	lx.Append(fx.Hook{
		OnStart: func(context.Context) (err error) {
			hw.Host, err = p2p.New(p.Ctx,
				maybePNet(p.PSK),
				p2p.Ping(false),
				p2p.NoListenAddrs, // also disables relay
				p2p.UserAgent("ww client"))
			return
		},
		OnStop: func(context.Context) error {
			return hw.Close()
		},
	})
}

func hookDiscover(lx fx.Lifecycle, host host.Host, d Discover) {
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ps, err := d.Discover(ctx)
			if err != nil {
				return errors.Wrap(err, "discover")
			}

			// TODO:  change this to an at-least-one-succeeds group
			var g errgroup.Group
			for _, pinfo := range ps {
				g.Go(connect(ctx, host, pinfo))
			}
			return g.Wait()
		},
	})
}

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}

func maybePNet(psk pnet.PSK) p2p.Option {
	if psk == nil {
		return p2p.ChainOptions()
	}

	return p2p.PrivateNetwork(psk)
}
