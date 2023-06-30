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

	"github.com/urfave/cli/v2"

	"github.com/wetware/ww"
	"github.com/wetware/ww/internal/cmd/run"
	cmdrun "github.com/wetware/ww/internal/cmd/run"
	"github.com/wetware/ww/internal/cmd/start"
	cmdstart "github.com/wetware/ww/internal/cmd/start"
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

var commands = []*cli.Command{
	cluster.Command(),
	debug.Command(),
	run.Command(),
	start.Command(),
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
		Commands: []*cli.Command{
			cmdrun.Command(),
			cmdstart.Command(),
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
