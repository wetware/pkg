package client

import (
	"context"
	"net"
	"strings"

	log "github.com/lthibault/log/pkg"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/client"

	logutil "github.com/wetware/ww/internal/util/log"
	wwclient "github.com/wetware/ww/pkg/client"
)

var (
	// initialized by `before` function
	logger log.Logger
	root   client.Client

	flags = []cli.Flag{
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
)

// Command constructor
func Command(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:        "client",
		Usage:       "interact with a live cluster",
		Flags:       flags,
		Before:      before(),
		After:       after(),
		Subcommands: subcommands(ctx),
	}
}

// before the wetware client
func before() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		logger = logutil.New(c)

		var d boot.Strategy
		switch {
		case c.StringSlice("join") != nil:
			d, err = join(c)
		case c.String("discover") != "":
			d, err = discoverPeers(c)
		default:
			err = errors.New("must specify either -join or -discover address")
		}

		if err == nil {
			root, err = client.Dial(context.Background(),
				wwclient.WithDiscover(d))
		}

		return
	}
}

func after() cli.AfterFunc {
	return func(c *cli.Context) error {
		return root.Close()
	}
}

func subcommands(ctx context.Context) []*cli.Command {
	return []*cli.Command{
		ls(ctx),
		subscribe(ctx),
		publish(ctx),
	}
}

func join(c *cli.Context) (as boot.StaticAddrs, err error) {
	as = make(boot.StaticAddrs, len(c.StringSlice("join")))
	for i, a := range c.StringSlice("join") {
		if as[i], err = multiaddr.NewMultiaddr(a); err != nil {
			break
		}
	}

	return
}

func discoverPeers(c *cli.Context) (boot.Strategy, error) {
	proto, param, err := head(c.String("discover"))
	if err != nil {
		return nil, err
	}

	switch proto {
	case "mdns":
		mdns := &boot.MDNS{Namespace: c.String("ns")}

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
