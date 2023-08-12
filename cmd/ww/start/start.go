package start

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	"github.com/wetware/pkg/server"
	"golang.org/x/exp/slog"
)

// Logger is used for logging by the RPC system. Each method logs
// messages at a different level, but otherwise has the same semantics:
//
//   - Message is a human-readable description of the log event.
//   - Args is a sequenece of key, value pairs, where the keys must be strings
//     and the values may be any type.
//   - The methods may not block for long periods of time.
//
// This interface is designed such that it is satisfied by *slog.Logger.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
	With(args ...any) *slog.Logger
}

var meta map[string]string

var flags = []cli.Flag{
	&cli.StringSliceFlag{
		Name:    "listen",
		Aliases: []string{"l"},
		Usage:   "host listen address",
		Value: cli.NewStringSlice(
			"/ip4/0.0.0.0/udp/0/quic",
			"/ip6/::0/udp/0/quic"),
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
					WithGroup(h.ID().String()).
					With(
						// "meta", meta,
						"peers", c.StringSlice("peer"),
						"discover", c.String("discover")),
			}

			err = config.Serve(c.Context, h)
			if err != context.Canceled {
				return err
			}

			return nil
		},
	}
}
