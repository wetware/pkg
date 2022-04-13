package client

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
	bootutil "github.com/wetware/ww/internal/util/boot"
	logutil "github.com/wetware/ww/internal/util/log"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
	"go.uber.org/fx"
)

var (
	app  *fx.App
	node *client.Node
)

func setup() cli.BeforeFunc {
	return func(c *cli.Context) error {
		app = fx.New(fx.NopLogger,
			fx.Supply(c),
			fx.Provide(
				localhost,
				logger,
				dialer),
			fx.Invoke(dial))

		ctx, cancel := context.WithTimeout(c.Context, c.Duration("timeout"))
		defer cancel()

		return app.Start(ctx)
	}
}

func teardown() cli.AfterFunc {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		return app.Stop(ctx)
	}
}

func logger(c *cli.Context) log.Logger {
	return logutil.New(c).With(discoveryFields(c))
}

func localhost(c *cli.Context, lx fx.Lifecycle) (host.Host, error) {
	h, err := libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(libp2pquic.NewTransport))
	if err == nil {
		lx.Append(closer(h))
	}

	return h, err
}

func dialer(c *cli.Context, h host.Host, lx fx.Lifecycle) (d client.Dialer, err error) {
	d.Vat = vat.Network{
		NS:   c.String("ns"),
		Host: h,
	}

	if c.IsSet("addr") {
		d.Boot, err = boot.NewStaticAddrStrings(c.StringSlice("addr")...)
		return
	}

	if c.String("discover") == "" {
		err = errors.New("must provide -discover or -addr flag")
		return
	}

	if d.Boot, err = bootutil.Dial(c, h); err == nil {
		lx.Append(closer(d.Boot.(io.Closer)))
	}

	return
}

func dial(d client.Dialer, lx fx.Lifecycle) {
	lx.Append(fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			node, err = d.Dial(ctx)
			return
		},
		OnStop: func(context.Context) error {
			return node.Close()
		},
	})
}

// discoveryFields reports the bootstrap multiaddr(s).
func discoveryFields(c *cli.Context) log.F {
	if c.String("discover") != "" {
		return log.F{"boot": c.String("discover")}
	}

	if len(c.StringSlice("addr")) > 0 {
		return log.F{"boot": c.StringSlice("addr")}
	}

	return nil
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: onclose(c),
	}
}

func onclose(c io.Closer) func(context.Context) error {
	return func(context.Context) error {
		return c.Close()
	}
}
