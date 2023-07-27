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
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Before: setup(),
		Action: start(),
	}
}

func start() cli.ActionFunc {
	return func(c *cli.Context) error {
		config := server.Config{
			Logger:   log.New(),
			NS:       c.String("ns"),
			Peers:    c.StringSlice("peer"),
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
