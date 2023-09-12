package run

import (
	"fmt"
	"os"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/vat"
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
	h, err := vat.DialP2P()
	if err != nil {
		return err
	}
	defer h.Close()

	bootstrap, err := newBootstrap(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}

	sess, err := vat.Dialer{
		Host:    h,
		Account: auth.SignerFromHost(h),
	}.DialDiscover(c.Context, bootstrap, c.String("ns"))
	if err != nil {
		return err
	}
	// defer sess.Terminate()

	// set up the local wetware environment.
	wetware := ww.Ww{
		NS:     c.String("ns"),
		Stdin:  c.App.Reader,
		Stdout: c.App.Writer,
		Stderr: c.App.ErrWriter,
		Root:   sess,
	}

	// fetch the ROM and run it
	rom, err := bytecode(c)
	if err != nil {
		return err
	}

	// Prepare argv for the process.
	args := []string{}
	if c.Args().Len() > 1 {
		args = append(args, c.Args().Slice()[1:]...)
		fmt.Println(args)
	}

	return wetware.Exec(c.Context, rom, csp.Args{
		Pid:  0,
		Ppid: 0,
		Cid:  rom.CID(),
		Cmd:  args,
	}.Encode()...)
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

func newBootstrap(c *cli.Context, h local.Host) (_ boot.Service, err error) {
	// use discovery service?
	if len(c.StringSlice("peer")) == 0 {
		serviceAddr := c.String("discover")
		return boot.DialString(h, serviceAddr)
	}

	// fast path; direct dial a peer
	maddrs := make([]ma.Multiaddr, len(c.StringSlice("peer")))
	for i, s := range c.StringSlice("peer") {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}
