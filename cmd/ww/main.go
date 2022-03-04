/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"os"
	"time"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"

	"github.com/wetware/ww/internal/cmd/client"
	"github.com/wetware/ww/internal/cmd/start"
	logutil "github.com/wetware/ww/internal/util/log"
	ww "github.com/wetware/ww/pkg"
)

var logger log.Logger

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
		Name:    "data",
		Usage:   "`path` to application data directory",
		Value:   "~/.ww",
		EnvVars: []string{"WW_DATA"},
	},
	// Statsd
	&cli.StringFlag{
		Name:        "statsd",
		Usage:       "send metrics to statsd daemon on `host:port`",
		EnvVars:     []string{"WW_STATSD"},
		DefaultText: "disabled",
	},
	&cli.StringFlag{
		Name:    "statsd-tagfmt",
		Usage:   "encode tags in influx or datadog `format`",
		Value:   "influx",
		EnvVars: []string{"WW_STATSD_TAGFMT"},
		Hidden:  true,
	},
	&cli.Float64Flag{
		Name:    "statsd-sample-rate",
		Usage:   "proportion of metrics to send",
		Value:   .1,
		EnvVars: []string{"WW_STATSD_SAMPLE_RATE"},
		Hidden:  true,
	},
	&cli.DurationFlag{
		Name:    "statsd-flush",
		Usage:   "buffer flush `interval` (0=disable)",
		Value:   time.Millisecond * 200,
		EnvVars: []string{"WW_STATSD_FLUSH"},
		Hidden:  true,
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
	client.Command(),
}

func before() cli.BeforeFunc {
	return func(c *cli.Context) error {
		logger = logutil.New(c)
		return nil
	}
}

func main() {
	run(&cli.App{
		Name:                 "wetware",
		Usage:                "the distributed programming language",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		Version:              ww.Version,
		EnableBashCompletion: true,
		Flags:                flags,
		Before:               before(),
		Commands:             commands,
		Metadata: map[string]interface{}{
			"version": ww.Version,
		},
	})
}

func run(app *cli.App) {
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
