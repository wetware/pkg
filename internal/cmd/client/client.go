package client

import (
	"context"
	"net"
	"strings"

	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	logutil "github.com/lthibault/wetware/internal/util/log"
	mautil "github.com/lthibault/wetware/pkg/util/multiaddr"

	"github.com/lthibault/wetware/pkg/client"
)

var cluster client.Client

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
	}
}

// Init the wetware client
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		log := logutil.New(c)

		var d client.Discover
		switch {
		case c.StringSlice("join") != nil:
			d, err = join(c)
		case c.String("discover") != "":
			d, err = discover(c, log)
		default:
			err = errors.New("must specify either -join or -discover address")
		}

		if err == nil {
			cluster, err = client.Dial(context.Background(), d,
				client.WithLogger(log))
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
	}}
}

func join(c *cli.Context) (client.StaticAddrs, error) {
	return mautil.NewMultiaddrs(c.StringSlice("join")...)
}

func discover(c *cli.Context, log log.Logger) (client.Discover, error) {
	proto, param, err := head(c.String("discover"))
	if err != nil {
		return nil, err
	}

	switch proto {
	case "mdns":
		switch mdns := new(client.MDNSDiscovery); param {
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
