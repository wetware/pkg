package run

import (
	"fmt"
	"os"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/urfave/cli/v2"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/util/proto"
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
		Name:   "run",
		Usage:  "execute a local webassembly process",
		Flags:  flags,
		Action: run,
	}
}

func run(c *cli.Context) error {
	h, err := client.DialP2P()
	if err != nil {
		return err
	}
	defer h.Close()

	d, err := boot.DialString(h, c.String("discover"))
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}

	boot := client.BootConfig{
		NS:         c.String("ns"),
		Host:       h,
		Discoverer: d,
	}

	// dial into the cluster
	dialer := client.Dialer[host.Host]{
		Bootstrapper: boot,
		Auth:         auth.AllowAll[host.Host],
		Opts:         nil, // TODO:  export something from the client side
	}

	sess, err := dialer.Dial(c.Context, addr(c, h))
	if err != nil {
		return err
	}
	defer sess.Close()

	// set up the local wetware environment.
	wetware := ww.Ww[host.Host]{
		NS:     c.String("ns"),
		Stdin:  c.App.Reader,
		Stdout: c.App.Writer,
		Stderr: c.App.ErrWriter,
		Client: sess.Client(),
	}

	// fetch the ROM and run it
	rom, err := bytecode(c)
	if err != nil {
		return err
	}

	return wetware.Exec(c.Context, rom)
}

func addr(c *cli.Context, h local.Host) *ww.Addr {
	return &ww.Addr{
		NS:    c.String("ns"),
		Peer:  h.ID(),
		Proto: proto.Namespace(c.String("ns")),
	}
}

func bytecode(c *cli.Context) (rom.ROM, error) {
	if c.Bool("stdin") {
		return rom.Read(c.App.Reader)
	}

	// file?
	if c.Args().Len() > 0 {
		return loadROM(c)
	}

	// use the default bytecode
	return rom.Default(), nil
}

func loadROM(c *cli.Context) (rom.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return rom.ROM{}, err
	}
	defer f.Close()

	return rom.Read(f)
}
