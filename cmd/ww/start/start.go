package start

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/libp2p/go-libp2p"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	"github.com/wetware/pkg/server"
)

var meta map[string]string

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip6/::0/udp/0/quic-v1"),
		EnvVars: []string{"WW_LISTEN"},
	},
	&cli.StringSliceFlag{
		Name:    "meta",
		Usage:   "metadata fields in key=value format",
		EnvVars: []string{"WW_META"},
	},
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "start a host process",
		Flags: flags,
		Before: func(c *cli.Context) error {
			// Parse and asign meta tags

			metaTags := c.StringSlice("meta")
			for _, tag := range metaTags {
				pair := strings.SplitN(tag, "=", 2)
				if len(pair) != 2 {
					return fmt.Errorf("invalid meta tag: %s", tag)
				}

				if meta == nil {
					meta = make(map[string]string, len(metaTags))
				}

				meta[pair[0]] = pair[1]
			}

			return nil
		},
		Action: func(c *cli.Context) error {
			// Configure a WASM runtime and execute a ROM.

			h, err := libp2p.New(
				libp2p.NoTransports,
				libp2p.Transport(tcp.NewTCPTransport),
				libp2p.Transport(quic.NewTransport),
				libp2p.ListenAddrStrings(c.StringSlice("listen")...))
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}
			defer h.Close()

			config := server.Config{
				NS:       c.String("ns"),
				Peers:    c.StringSlice("peer"),
				Discover: c.String("discover"),
				Meta:     meta,
				Logger: slog.Default().
					With(
						"id", h.ID(),
						/*"peers", c.StringSlice("peer"),*/
						/*"discover", c.String("discover")*/),
			}

			err = config.Serve(c.Context, h)
			if err != context.Canceled {
				return err
			}

			return nil
		},
	}
}
