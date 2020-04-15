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
