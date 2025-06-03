/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/tetratelabs/wazero/sys"
	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/cmd/ww/benchmark"
	"github.com/wetware/pkg/cmd/ww/cluster"
	"github.com/wetware/pkg/cmd/ww/ls"
	"github.com/wetware/pkg/cmd/ww/ps"
	"github.com/wetware/pkg/cmd/ww/run"
	"github.com/wetware/pkg/cmd/ww/start"
	"github.com/wetware/pkg/util/proto"
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
	ls.Command(),
	ps.Command(),
	run.Command(),
	start.Command(),
	cluster.Command(),
	benchmark.Command(),
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
		Version:              proto.Version,
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		EnableBashCompletion: true,
		Flags:                flags,
		Before:               setup,
		Commands:             commands,
	}

	die(app.RunContext(ctx, os.Args))
}

func setup(c *cli.Context) error {
	var h = &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// enable debug logs?
	if c.Bool("debug") {
		h.Level = slog.LevelDebug
	}

	// enable json logging?
	if c.Bool("json") {
		handler := slog.NewJSONHandler(os.Stderr, h)
		slog.SetDefault(slog.New(handler))
	} else {
		slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level:      h.Level,
			TimeFormat: time.Kitchen,
			NoColor:    colorDisabled(),
		})))
	}

	return nil
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
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}

func colorDisabled() bool {
	return !isatty.IsTerminal(os.Stderr.Fd())
}

func bootstrapAddr() string {
	return path.Join("/ip4/228.8.8.8/udp/8822/multicast", eth())
}

//	func bootstrapAddr() string {
//		return path.Join("/ip4/228.8.8.8/udp/8822/multicast", loopback())
//	}
func eth() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		if runtime.GOARCH == "amd64" {
			hostname, err := os.Hostname()
			if err != nil || hostname != "labtop" {
				return "enp7s0"
			} else {
				return "enp58s0f1"
			}
		} else {
			return "enxb827eb6d6e99"
		}
	}
}

func loopback() string {
	switch runtime.GOOS {
	case "darwin":
		return "lo0"
	default:
		return "lo"
	}
}
