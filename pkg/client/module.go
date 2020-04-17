package client

import (
	"context"
	"time"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

func module(c *Client, d Discover, opt []Option) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(opt, struct{ Discover }{d}),
		fx.Provide(
			newCtx,
			newConfig,
			newHost,
			newClient,
		),
		fx.Invoke(join),
		fx.Populate(c),
	)
}

func newClient(host host.Host) Client {
	return Client{
		host: host,
	}
}

func newCtx(lx fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	lx.Append(fx.Hook{
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})

	return ctx
}

func join(ctx context.Context, host host.Host, d struct{ Discover }) error {
	ps, err := d.DiscoverPeers(ctx)
	if err != nil {
		return errors.Wrap(err, "discover")
	}

	// TODO:  change this to an at-least-one-succeeds group
	var g errgroup.Group
	for _, pinfo := range ps {
		g.Go(connect(ctx, host, pinfo))
	}

	return errors.Wrap(g.Wait(), "join")
}

func connect(ctx context.Context, host host.Host, pinfo peer.AddrInfo) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		return host.Connect(ctx, pinfo)
	}
}
