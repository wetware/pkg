package start

import (
	"fmt"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/auth"
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
		Name:   "start",
		Usage:  "start a host process",
		Flags:  flags,
		Action: serve,
	}
}

func serve(c *cli.Context) error {
	h, err := server.ListenP2P(c.StringSlice("listen")...)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer h.Close()

	d, err := boot.ListenString(h, c.String("discover"))
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}

	ns := boot.Namespace{
		Name:      c.String("ns"),
		Bootstrap: d,
		Ambient:   d,
	}

	boot := server.BootConfig{
		Net:   ns,
		Host:  h,
		Peers: c.StringSlice("peer"),
		RPC:   nil, // server doesn't export a capabiltity (yet)
	}

	dialer := client.Config[host.Host]{
		PeerDialer: boot,
		Auth:       auth.AllowAll[host.Host],
	}

	export := server.Config{
		NS:        c.String("ns"),
		Meta:      c.StringSlice("meta"),
		Host:      h,
		Discovery: d,
	}

	return ww.Vat[host.Host]{
		Addr:   addr(c, h),
		Host:   h,
		Dialer: dialer,
		Export: export,
	}.ListenAndServe(c.Context)
}

// local address on
func addr(c *cli.Context, h local.Host) *ww.Addr {
	return &ww.Addr{
		NS:  c.String("ns"),
		Vat: h.ID(),
	}
}
