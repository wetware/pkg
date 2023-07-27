package run

import (
	"os"
	"path"
	"runtime"

	"capnproto.org/go/capnp/v3"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringSliceFlag{
		Name:    "dial",
		Usage:   "multiaddr of server node",
		EnvVars: []string{"WW_DIAL"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "multiaddr of peer-discovery service",
		Value:   bootstrapAddr(),
		EnvVars: []string{"WW_DISCOVER"},
	},
	&cli.StringFlag{
		Name:    "rom",
		Usage:   "cid of boot rom",
		EnvVars: []string{"WW_ROM"},
	},
	&cli.BoolFlag{
		Name:    "stdin",
		Aliases: []string{"s"},
		Usage:   "load system image from stdin",
	},
	&cli.BoolFlag{
		Name:  "debug",
		Usage: "enable debug logging",
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
		rom, err := bytecode(c)
		if err != nil {
			return err
		}

		return ww.Ww{
			NS:     c.String("ns"),
			Stdin:  c.App.Reader,
			Stdout: c.App.Writer,
			Stderr: c.App.ErrWriter,
			Client: capnp.Client{}, // TODO:  Host client goes here
		}.Exec(c.Context, rom)
	}
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

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
