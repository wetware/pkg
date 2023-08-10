/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/rom"
	"github.com/wetware/pkg/rom/ls"
)

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "ns",
		Usage:   "cluster namespace",
		Value:   "ww",
		EnvVars: []string{"WW_NS"},
	},
	&cli.StringSliceFlag{
		Name:    "peer",
		Aliases: []string{"p"},
		Usage:   "bootstrap peer `ADDR`",
		EnvVars: []string{"WW_PEERS"},
	},
	&cli.StringFlag{
		Name:    "discover",
		Aliases: []string{"d"},
		Usage:   "use discovery service",
		Value:   bootstrapAddr(),
		EnvVars: []string{"WW_DISCOVER"},
	},

	// Category:  Logging
	&cli.BoolFlag{
		Name:     "debug",
		Usage:    "enable debug logging",
		EnvVars:  []string{"WW_DEBUG"},
		Category: "LOGGING",
	},
	&cli.BoolFlag{
		Name:     "json",
		Usage:    "enable json logging",
		EnvVars:  []string{"WW_JSON"},
		Category: "LOGGING",
	},
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGKILL)
	defer cancel()

	app := &cli.App{
		Name:                 "wetware",
		Version:              ww.Version,
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		EnableBashCompletion: true,
		Flags:                flags,
		Action: func(c *cli.Context) error {
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
		},
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1) // application error
	}
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

func dialAndExec(c *cli.Context, wetware ww.Ww, rom ww.ROM) error {
	h, err := clientHost(c)
	if err != nil {
		return err
	}
	defer h.Close()

	conn, err := client.Dialer{
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
	// is it a subcommand?
	if rom, ok := subcommand(c); ok {
		return rom, nil
	}

	// stdin?
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

func subcommand(c *cli.Context) (ww.ROM, bool) {
	switch c.Args().First() {
	case "ls":
		return ls.ROM(), true
	}

	log.Warn(c.Args())
	return ww.ROM{}, false
}

func loadROM(c *cli.Context) (ww.ROM, error) {
	f, err := os.Open(c.Args().First())
	if err != nil {
		return ww.ROM{}, err
	}
	defer f.Close()

	return ww.Read(f)
}
