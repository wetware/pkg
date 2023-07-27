package run

import (
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww"
	"github.com/wetware/ww/client"
)

var flags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "join",
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
		Name:   "run",
		Usage:  "execute a local webassembly process",
		Flags:  flags,
		Action: run(),
	}
}

func run() cli.ActionFunc {
	return func(c *cli.Context) error {
		wetware := ww.Ww{
			NS:     c.String("ns"),
			Stdin:  c.App.Reader,
			Stdout: c.App.Writer,
			Stderr: c.App.ErrWriter,
		}

		rom, err := bytecode(c)
		if err != nil {
			return err
		}

		// dial into a cluster?
		if c.Bool("dial") {
			return dialAndExec(c, wetware, rom)
		}

		// run without connecting to a cluster
		return wetware.Exec(c.Context, rom)
	}
}

func dialAndExec(c *cli.Context, wetware ww.Ww, rom ww.ROM) error {
	h, err := clientHost(c)
	if err != nil {
		return err
	}
	defer h.Close()

	conn, err := client.Config{
		Logger:   log.New(),
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
	}.Dial(c.Context, h)
	if err != nil {
		return err
	}
	defer conn.Close()

	wetware.Client = conn.Bootstrap(c.Context)
	return wetware.Exec(c.Context, rom)
}

func clientHost(c *cli.Context) (host.Host, error) {
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
	return ww.DefaultROM(), nil
}

func loadROM(c *cli.Context) (ww.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return ww.ROM{}, err
	}
	defer f.Close()

	return ww.Read(f)
}
