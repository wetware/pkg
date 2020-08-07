package clientutil

import (
	"context"
	"net"
	"strings"

	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/client"
)

// Dial into a cluster with CLI args
//
// c must contain either a -join or -discover flag.
func Dial(ctx context.Context, c *cli.Context) (root client.Client, err error) {
	var d boot.Strategy
	switch {
	case c.StringSlice("join") != nil:
		d, err = Join(c)
	case c.String("discover") != "":
		d, err = Bootstrap(c)
	default:
		err = errors.New("must specify either -join or -discover address")
	}

	if err == nil {
		root, err = client.Dial(ctx, client.WithStrategy(d))
	}

	return
}

// Join addrs from CLI context.
func Join(c *cli.Context) (as boot.StaticAddrs, err error) {
	as = make(boot.StaticAddrs, len(c.StringSlice("join")))
	for i, a := range c.StringSlice("join") {
		if as[i], err = multiaddr.NewMultiaddr(a); err != nil {
			break
		}
	}

	return
}

// Bootstrap strategy from CLI context.
func Bootstrap(c *cli.Context) (boot.Strategy, error) {
	proto, param, err := head(c.String("discover"))
	if err != nil {
		return nil, err
	}

	switch proto {
	case "mdns":
		mdns := &boot.MDNS{Namespace: c.String("ns")}

		switch param {
		case "":
			return mdns, nil
		default:
			if mdns.Interface, err = net.InterfaceByName(param); err != nil {
				return nil, errors.Wrap(err, "discover mdns")
			}

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
