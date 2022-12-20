/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"os"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"

	"github.com/wetware/ww/internal/cmd/cluster"
	"github.com/wetware/ww/internal/cmd/debug"
	"github.com/wetware/ww/internal/cmd/start"
	ww "github.com/wetware/ww/pkg"
)

var flags = []cli.Flag{
	// Logging
	&cli.StringFlag{
		Name:    "logfmt",
		Aliases: []string{"f"},
		Usage:   "`format` logs as text, json or none",
		Value:   "text",
		EnvVars: []string{"WW_LOGFMT"},
	},
	&cli.StringFlag{
		Name:    "loglvl",
		Usage:   "set logging `level` to trace, debug, info, warn, error or fatal",
		Value:   "info",
		EnvVars: []string{"WW_LOGLVL"},
	},
	&cli.PathFlag{
		Name:        "data",
		Usage:       "persist cache data to `path`",
		DefaultText: "disabled",
		EnvVars:     []string{"WW_DATA"},
	},
	// Statsd
	&cli.StringFlag{
		Name:        "metrics",
		Aliases:     []string{"statsd"},
		Usage:       "send metrics to udp `host:port`",
		EnvVars:     []string{"WW_METRICS", "WW_STATSD"},
		DefaultText: "disabled",
	},
	// Misc.
	&cli.BoolFlag{
		Name:    "prettyprint",
		Aliases: []string{"pp"},
		Usage:   "pretty-print JSON output",
		Hidden:  true,
	},
}

var commands = []*cli.Command{
	start.Command(),
	cluster.Command(),
	debug.Command(),
}

func main() {
	run(&cli.App{
		Name:                 "wetware",
		HelpName:             "ww",
		Usage:                "simple, secure clusters",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		Version:              ww.Version,
		EnableBashCompletion: true,
		Flags:                flags,
		Commands:             commands,
		Metadata: map[string]interface{}{
			"version": ww.Version,
		},
	})
}

func run(app *cli.App) {
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
