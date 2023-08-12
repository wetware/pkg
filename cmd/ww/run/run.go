package run

import (
	"errors"
	"os"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/system"
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
	return &cli.Command{
		Name:  "run",
		Usage: "execute a local webassembly process",
		Flags: flags,
		Action: func(c *cli.Context) error {
			h, err := clientHost(c)
			if err != nil {
				return err
			}
			defer h.Close()

			// dial into the cluster;  if -dial=false, client is null.
			client, err := dial[host.Host](c, h)
			if err != nil {
				return err
			}
			defer client.Release()

			// set up the local wetware environment.
			wetware := ww.Ww[host.Host]{
				NS:     c.String("ns"),
				Stdin:  c.App.Reader,
				Stdout: c.App.Writer,
				Stderr: c.App.ErrWriter,
				Client: client,
			}

			// fetch the ROM and run it
			rom, err := bytecode(c)
			if err != nil {
				return err
			}

			return wetware.Exec(c.Context, rom)
		},
	}
}

func dial[T ~capnp.ClientKind](c *cli.Context, h local.Host) (T, error) {
	// dial into a cluster?
	if c.Bool("dial") {
		return system.Boot[T](c, h)
	}

	// we're not connecting to the cluster
	return failure[T](errors.New("disconnected"))
}

func failure[T ~capnp.ClientKind](err error) (T, error) {
	return T(capnp.ErrorClient(err)), err
}

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func bytecode(c *cli.Context) (ww.ROM, error) {
	if c.Bool("stdin") {
		return ww.Read(c.App.Reader)
	}

	// file?
	if c.Args().Len() > 0 {
		return loadROM(c)
	}

	// use the default bytecode
	return rom.Default(), nil
}

func loadROM(c *cli.Context) (ww.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return ww.ROM{}, err
	}
	defer f.Close()

	return ww.Read(f)
}