package client

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	bootutil "github.com/wetware/ww/internal/util/boot"
	"github.com/wetware/ww/pkg/client"
	"go.uber.org/multierr"
)

var (
	h    host.Host
	node *client.Node
)

func dial() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		h, err = libp2p.New(
			libp2p.NoTransports,
			libp2p.NoListenAddrs,
			libp2p.Transport(libp2pquic.NewTransport))
		if err == nil {
			logger.With(discoveryFields(c)).Debug("dialing cluster")
			node, err = bootutil.Dial(c, h)
		}

		return
	}
}

func shutdown() cli.AfterFunc {
	return func(c *cli.Context) (err error) {
		if node != nil {
			err = node.Close()
		}

		if h != nil {
			err = multierr.Append(err, h.Close())
		}

		return
	}
}

// discoveryFields reports the bootstrap multiaddr(s).
func discoveryFields(c *cli.Context) log.F {
	if c.String("discover") != "" {
		return log.F{"boot": c.String("discover")}
	}

	if len(c.StringSlice("addr")) > 0 {
		return log.F{"boot": c.StringSlice("addr")}
	}

	return nil
}
