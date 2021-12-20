/*
	Wetware - the distributed programming language
	Copyright 2020, Louis Thibault.  All rights reserved.
*/

package main

import (
	"os"

	"github.com/urfave/cli/v2"

	"github.com/lthibault/log"

	"github.com/wetware/ww/internal/cmd/client"
	"github.com/wetware/ww/internal/cmd/start"
	logutil "github.com/wetware/ww/internal/util/log"
)

const version = "0.0.0"

var logger log.Logger

var flags = []cli.Flag{
	&cli.StringFlag{
		Name:    "logfmt",
		Aliases: []string{"f"},
		Usage:   "text, json, none",
		Value:   "text",
		EnvVars: []string{"WW_LOGFMT"},
	},
	&cli.StringFlag{
		Name:    "loglvl",
		Usage:   "trace, debug, info, warn, error, fatal",
		Value:   "info",
		EnvVars: []string{"WW_LOGLVL"},
	},
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
	// discover.Command(),
	// shell.Command(),
	// keygen.Command(),
	// boot.Command(),
}

func main() {
	run(&cli.App{
		Name:                 "wetware",
		Usage:                "the distributed programming language",
		UsageText:            "ww [global options] command [command options] [arguments...]",
		Copyright:            "2020 The Wetware Project",
		Version:              version,
		EnableBashCompletion: true,
		Flags:                flags,
		Commands:             commands,
		Before:               before(),
		Metadata: map[string]interface{}{
			"version": version,
		},
	})
}

func before() cli.BeforeFunc {
	return func(c *cli.Context) error {
		logger = logutil.New(c)
		for _, set := range []func(log.Logger){
			start.SetLogger,
			client.SetLogger,
			// discover.SetLogger,
			// shell.SetLogger,
			// keygen.SetLogger,
			// boot.SetLogger,
		} {
			set(logger)
		}

		return nil
	}
}

func run(app *cli.App) {
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
