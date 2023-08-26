package start

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/server"
)

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
		Action: func(c *cli.Context) error {
			h, err := libp2p.New(
				libp2p.NoTransports,
				libp2p.Transport(tcp.NewTCPTransport),
				libp2p.Transport(quic.NewTransport),
				libp2p.ListenAddrStrings(c.StringSlice("listen")...))
			if err != nil {
				return fmt.Errorf("listen: %w", err)
			}
			defer h.Close()

			discovery, err := newDiscoveryService(c, h)
			if err != nil {
				return fmt.Errorf("discovery: %w", err)
			}

			boot := boot.Config{
				Peers:     c.StringSlice("peer"),
				Discovery: discovery,
			}

			dialer := client.Config{
				PeerDialer: boot,
			}

			export := server.Config{
				NS:        c.String("ns"),
				Host:      h,
				Meta:      c.StringSlice("meta"),
				Discovery: discovery,
			}

			return ww.Vat[host.Host]{
				Addr:   addr(c, h),
				Host:   h,
				Dialer: dialer,
				Export: export,
			}.ListenAndServe(c.Context)
		},
	}
}

// local address on
func addr(c *cli.Context, h local.Host) *ww.Addr {
	return &ww.Addr{
		NS:  c.String("ns"),
		Vat: h.ID(),
	}
}

func newDiscoveryService(c *cli.Context, h local.Host) (boot.Service, error) {
	return boot.ListenString(h, c.String("discover"))
}
