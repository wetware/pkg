package run

import (
	"os"

	"capnproto.org/go/capnp/v3/rpc"
	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/rom"
	"golang.org/x/exp/slog"
)

var flags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "dial",
		Usage:   "connect to cluster",
		EnvVars: []string{"WW_DIAL"},
	},
	&cli.BoolFlag{
		Name:     "stdin",
		Aliases:  []string{"s"},
		Usage:    "load system image from stdin",
		Category: "ROM",
	},
}

func Command() *cli.Command {
	rom := rom.Default()

	return &cli.Command{
		Name:  "run",
		Usage: "execute a local webassembly process",
		Flags: flags,
		Before: func(c *cli.Context) (err error) {
			if c.Bool("stdin") {
				rom, err = ww.Read(c.App.Reader)
			} else if c.Args().Len() > 0 {
				rom, err = loadROM(c)
			}

			return
		},
		Action: func(c *cli.Context) error {
			h, err := client.NewHost()
			if err != nil {
				return err
			}
			defer h.Close()

			conn, err := dial(c, h)
			if err != nil {
				return err
			}
			defer conn.Close()

			export, err := bootstrap(c, conn)
			if err != nil {
				return err
			}
			defer export.Release()

			// set up the local wetware environment.
			return ww.Ww[host.Host]{
				NS:              c.String("ns"),
				Stdin:           c.App.Reader,
				Stdout:          c.App.Writer,
				Stderr:          c.App.ErrWriter,
				BootstrapClient: export,
			}.Exec(c.Context, rom)
		},
	}
}

func bootstrap(c *cli.Context, conn *rpc.Conn) (host.Host, error) {
	client := conn.Bootstrap(c.Context)
	return host.Host(client), client.Resolve(c.Context)
}

func dial(c *cli.Context, h local.Host) (*rpc.Conn, error) {
	return client.Dial(c.Context, h, &client.DialConfig{
		Logger:   slog.Default().WithGroup("local"),
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
	})
}

func loadROM(c *cli.Context) (ww.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return ww.ROM{}, err
	}
	defer f.Close()

	return ww.Read(f)
}
