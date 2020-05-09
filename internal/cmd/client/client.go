package client

import (
	"context"
	"net"
	"strings"

	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	logutil "github.com/lthibault/wetware/internal/util/log"
	"github.com/lthibault/wetware/pkg/client"
	mautil "github.com/lthibault/wetware/pkg/util/multiaddr"

	discover "github.com/lthibault/wetware/pkg/discover"
)

var (
	cluster client.Client
	logger  log.Logger
	proc    = ctxutil.WithLifetime(context.Background())
)

// Flags for the `start` command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "join",
			Aliases: []string{"j"},
			Usage:   "connect to cluster through specified peers",
			EnvVars: []string{"WW_JOIN"},
		},
		&cli.StringFlag{
			Name:    "discover",
			Aliases: []string{"d"},
			Usage:   "automatic peer discovery settings",
			Value:   "/mdns",
			EnvVars: []string{"WW_DISCOVER"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Aliases: []string{"ns"},
			Usage:   "cluster namespace (must match dial host)",
			Value:   "ww",
			EnvVars: []string{"WW_NAMESPACE"},
		},
	}
}

// Init the wetware client
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		logger = logutil.New(c)

		var d discover.Strategy
		switch {
		case c.StringSlice("join") != nil:
			d, err = join(c)
		case c.String("discover") != "":
			d, err = discoverPeers(c, logger)
		default:
			err = errors.New("must specify either -join or -discover address")
		}

		if err == nil {
			cluster, err = client.Dial(context.Background(),
				client.WithDiscover(d),
				client.WithLogger(logger))
		}

		return
	}
}

// Shutdown the wetware client
func Shutdown() cli.AfterFunc {
	return func(c *cli.Context) error {
		return cluster.Close()
	}
}

// Commands under `client`
func Commands() []*cli.Command {
	return []*cli.Command{{
		Name:      "ls",
		Usage:     "list cluster elements",
		ArgsUsage: "path",
		Flags:     lsFlags(),
		Action:    lsAction(),
	}, {
		Name:    "subscribe",
		Aliases: []string{"sub"},
		Flags:   subFlags(),
		Action:  subAction(),
	}, {
		Name:    "publish",
		Aliases: []string{"pub"},
		Flags:   pubFlags(),
		Action:  pubAction(),
	}}
}

func join(c *cli.Context) (discover.StaticAddrs, error) {
	return mautil.NewMultiaddrs(c.StringSlice("join")...)
}

func discoverPeers(c *cli.Context, log log.Logger) (discover.Strategy, error) {
	proto, param, err := head(c.String("discover"))
	if err != nil {
		return nil, err
	}

	switch proto {
	case "mdns":
		mdns := &discover.MDNS{Namespace: c.String("ns")}

		switch param {
		case "":
			log.Debug("using default multicast interface")
			return mdns, nil
		default:
			if mdns.Interface, err = net.InterfaceByName(param); err != nil {
				return nil, errors.Wrap(err, "discover mdns")
			}

			log.Debugf("using multicast interface %s", param)
			return mdns, nil
		}
	default:
		return nil, errors.Errorf("unknown discovery protocol %s", proto)
	}
}

func head(s string) (head string, body string, err error) {
	switch ss := strings.Split(strings.Trim(s, "/"), "/"); len(ss) {
	case 0:
		err = errors.New("invalid discovery addr")
	case 1:
		head = ss[0]
	default:
		head = ss[0]
		body = strings.Join(ss[1:], "/")
	}

	return
}
