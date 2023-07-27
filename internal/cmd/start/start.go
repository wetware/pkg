package start

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/server"
)

var meta map[string]string

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "start a host process",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "ns",
				Usage:   "cluster namespace",
				Value:   "ww",
				EnvVars: []string{"WW_NS"},
			},
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
				Name:    "join",
				Aliases: []string{"j"},
				Usage:   "join cluster via existing peer `ADDR`",
				EnvVars: []string{"WW_JOIN"},
			},
			&cli.StringFlag{
				Name:    "discover",
				Aliases: []string{"d"},
				Usage:   "multiaddr of peer-discovery service",
				Value:   bootstrapAddr(),
				EnvVars: []string{"WW_DISCOVER"},
			},
			&cli.StringSliceFlag{
				Name:    "meta",
				Usage:   "metadata fields in key=value format",
				EnvVars: []string{"WW_META"},
			},
		},
		Before: setup(),
		Action: start(),
	}
}

func start() cli.ActionFunc {
	return func(c *cli.Context) error {
		config := server.Config{
			Logger:   log.New(),
			NS:       c.String("ns"),
			Join:     c.StringSlice("join"),
			Discover: c.String("discover"),
			Meta:     meta,
		}

		err := config.ListenAndServe(c.Context, c.StringSlice("listen")...)
		if err != context.Canceled {
			return err
		}

		return nil
	}
}

func setup() cli.BeforeFunc {
	return func(c *cli.Context) error {
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
	}
}

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
