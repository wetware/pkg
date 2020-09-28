package boot

import (
	"context"
	"encoding/json"
	"net"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	ctxutil "github.com/wetware/ww/internal/util/ctx"
	logutil "github.com/wetware/ww/internal/util/log"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/boot"
)

var (
	// initialized by `before` function
	logger ww.Logger
	d      boot.Strategy

	ctx = ctxutil.WithDefaultSignals(context.Background())

	flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "protocol",
			Aliases: []string{"p"},
			Usage:   "peer discovery protocol",
			Value:   "mdns",
		},
		&cli.DurationFlag{
			Name:    "timeout",
			Aliases: []string{"t"},
			Usage:   "time to wait for cluster response",
		},
		&cli.BoolFlag{
			Name:    "prettyprint",
			Aliases: []string{"pretty", "pp"},
			Usage:   "indent JSON output",
		},
		&cli.IntFlag{
			Name:    "number",
			Aliases: []string{"n"},
			Usage:   "number of records to return (0 = stream)",
			Value:   1,
		},
	}
)

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "discover",
		Usage:  "discover peers on the network",
		Flags:  flags,
		Before: before(),
		Action: run(),
	}
}

func before() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		logger = logutil.New(c)

		switch c.String("protocol") {
		case "mdns":
			mdns := new(boot.MDNS)
			if name := c.String("if"); name != "" {
				if mdns.Interface, err = net.InterfaceByName(name); err != nil {
					return errors.Wrap(err, "interface")
				}
			}

			d = mdns
		default:
			err = errors.Errorf("unknown discovery protocol %s", c.String("protocol"))
		}

		return
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		var cancel context.CancelFunc
		if c.Duration("timeout") != 0 {
			ctx, cancel = context.WithTimeout(ctx, c.Duration("timeout"))
			defer cancel()
		}

		enc := json.NewEncoder(c.App.Writer)
		if c.Bool("prettyprint") {
			enc.SetIndent("", "  ")
		}

		peers, err := d.DiscoverPeers(ctx,
			// discover.WithLogger(logger),
			boot.WithLimit(c.Int("n")))
		if err != nil {
			return err
		}

		for info := range peers {
			if err = enc.Encode(info); err != nil {
				return err
			}
		}

		return nil
	}
}
