/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"context"
	_ "embed"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"

	"github.com/tetratelabs/wazero/sys"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/internal/cluster"
	"github.com/wetware/pkg/internal/ls"
	"github.com/wetware/pkg/internal/run"
	"github.com/wetware/pkg/internal/start"
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

	log := slog.Default()

	app := &cli.App{
		Name:                 "wetware",
		Version:              ww.Version,
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		EnableBashCompletion: true,
		Flags:                flags,
		Before: func(c *cli.Context) error {
			log = log.With(
				"version", ww.Version,
				"ns", c.String("ns"))
			return nil
		},
		Commands: []*cli.Command{
			ls.Command(log),
			run.Command(log),
			start.Command(log),
			cluster.Command(log),
		},
	}

	die(app.RunContext(ctx, os.Args))
}

func die(err error) {
	if e, ok := err.(*sys.ExitError); ok {
		switch e.ExitCode() {
		case sys.ExitCodeContextCanceled:
			err = context.Canceled
		case sys.ExitCodeDeadlineExceeded:
			err = context.DeadlineExceeded
		default:
			os.Exit(int(e.ExitCode()))
		}
	}

	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
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
