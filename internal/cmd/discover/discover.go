package discover

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/discovery"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/wetware/casm/pkg/boot"

	logutil "github.com/wetware/ww/internal/util/log"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "casm",
		EnvVars: []string{"CASM_NS"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "discovery service multiaddress",
		Value:   "/multicast/ip4/228.8.8.8/udp/8822",
	},
	&cli.DurationFlag{
		Name:    "timeout",
		Aliases: []string{"t"},
		Usage:   "stop after t seconds",
	},
	&cli.IntFlag{
		Name:    "number",
		Aliases: []string{"n"},
		Usage:   "number of records to return (0 = stream)",
		Value:   1,
	},
}

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "discover",
		Usage:  "discover peers on the network",
		Flags:  flags,
		Action: discover(),
	}
}

func discover() cli.ActionFunc {
	return func(c *cli.Context) error {
		var (
			enc *json.Encoder
			d   discovery.Discoverer
		)

		app := fx.New(fx.NopLogger,
			fx.Supply(c),
			fx.Populate(&enc, &d),
			fx.Provide(
				newEncoder,
				newTransport,
				newDiscoveryClient))

		if err := app.Start(c.Context); err != nil {
			return err
		}

		ctx, cancel := maybeTimeout(c)
		defer cancel()

		if c.Duration("t") > 0 {
			ctx, cancel = context.WithTimeout(c.Context, c.Duration("t"))
			defer cancel()
		}

		ps, err := d.FindPeers(ctx, c.String("ns"), discovery.Limit(c.Int("n")))
		if err != nil {
			return err
		}

		for {
			select {
			case info := <-ps:
				if err := enc.Encode(info); err != nil {
					return err
				}

			case <-app.Done():
				return shutdown(app)
			}
		}
	}
}

func shutdown(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	return app.Stop(ctx)
}

func maybeTimeout(c *cli.Context) (context.Context, context.CancelFunc) {
	if c.Duration("t") > 0 {
		return context.WithTimeout(c.Context, c.Duration("t"))
	}

	return c.Context, func() {}
}

func newEncoder(c *cli.Context) *json.Encoder {
	enc := json.NewEncoder(c.App.Writer)
	if c.Bool("prettyprint") {
		enc.SetIndent("", "  ")
	}

	return enc
}

func newDiscoveryClient(c *cli.Context, t boot.Transport, lx fx.Lifecycle) (discovery.Discoverer, error) {
	d, err := boot.NewMulticastClient(
		boot.WithTransport(t),
		boot.WithLogger(logutil.New(c)))

	if err == nil {
		lx.Append(closer(d))
	}

	return d, err
}

func newTransport(c *cli.Context) (boot.Transport, error) {
	m, err := ma.NewMultiaddr(c.String("d"))
	if err != nil {
		return nil, fmt.Errorf("%w:  %s", err, m)
	}

	return boot.NewTransport(m)
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error { return c.Close() },
	}
}
