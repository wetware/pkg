package main

import (
	"os"

	"github.com/urfave/cli/v2"

	log "github.com/lthibault/log/pkg"

	"github.com/lthibault/wetware/internal/cmd/client"
	"github.com/lthibault/wetware/internal/cmd/keygen"
	"github.com/lthibault/wetware/internal/cmd/repo"
	"github.com/lthibault/wetware/internal/cmd/start"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "logfmt",
				Aliases: []string{"f"},
				Usage:   "text, json, none",
				Value:   "text",
				EnvVars: []string{"CASM_LOGFMT"},
			},
			&cli.StringFlag{
				Name:    "loglvl",
				Usage:   "trace, debug, info, warn, error, fatal",
				Value:   "info",
				EnvVars: []string{"CASM_LOGLVL"},
			},

			/************************
			*	undocumented flags	*
			*************************/
			&cli.BoolFlag{
				Name:    "prettyprint",
				Aliases: []string{"pp"},
				Usage:   "pretty-print JSON output",
				Hidden:  true,
			},
		},
		Commands: []*cli.Command{{
			Name:   "start",
			Usage:  "start a host process",
			Flags:  start.Flags(),
			Action: start.Run(),
		}, {
			Name:        "repo",
			Usage:       "create or configure hosts",
			Subcommands: repo.Commands(),
		}, {
			Name:        "client",
			Usage:       "interact with a live cluster",
			Flags:       client.Flags(),
			Before:      client.Init(),
			Subcommands: client.Commands(),
		}, {
			Name:        "keygen",
			Usage:       "generate a shared secret for a cluster",
			Description: keygen.Description,
			Flags:       keygen.Flags(),
			Action:      keygen.Run(),
		}},
	}

	if err := app.Run(os.Args); err != nil {
		log.New().Fatal(err)
	}
}
