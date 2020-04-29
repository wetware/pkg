package discover

import (
	"context"
	"encoding/json"
	"net"
	"time"

	discover "github.com/lthibault/wetware/pkg/discover"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var d discover.Strategy

// Init the discovery service
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
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
			Value:   time.Second * 5,
		},
		&cli.BoolFlag{
			Name:    "prettyprint",
			Aliases: []string{"pretty", "pp"},
			Usage:   "indent JSON output",
		},
		&cli.IntFlag{
			Name:    "number",
			Aliases: []string{"n"},
			Usage:   "max number of peers to return",
			Hidden:  true, // TODO:  implement multiple results in client.MDNSDiscovery
		},
	}
}

// Run the `discover` command.
func Run() cli.ActionFunc {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), c.Duration("timeout"))
		defer cancel()

		ps, err := d.DiscoverPeers(ctx)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(c.App.Writer)
		if c.Bool("prettyprint") {
			enc.SetIndent("", "  ")
		}

		for _, info := range ps {
			if err = enc.Encode(info); err != nil {
				break
			}
		}

		return err
	}
}
