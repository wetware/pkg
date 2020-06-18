package discover

import (
	"context"
	"encoding/json"
	"net"

	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	logutil "github.com/lthibault/wetware/internal/util/log"
	discover "github.com/lthibault/wetware/pkg/discover"
)

var (
	d      discover.Strategy
	logger log.Logger
	ctx    = ctxutil.WithDefaultSignals(context.Background())
)

// Init the discovery service
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		logger = logutil.New(c)

		switch c.String("protocol") {
		case "mdns":
			mdns := new(discover.MDNS)
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

// Flags for `discover` command.
func Flags() []cli.Flag {
	return []cli.Flag{
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
}

// Run the `discover` command.
func Run() cli.ActionFunc {
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
			discover.WithLogger(logger),
			discover.WithLimit(c.Int("n")))
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
