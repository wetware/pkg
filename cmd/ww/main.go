package main

import (
	"os"

	"github.com/urfave/cli/v2"

	log "github.com/lthibault/log/pkg"
	"github.com/lthibault/wetware/internal/cmd/start"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{{
			Name:   "start",
			Usage:  "start a host process",
			Flags:  start.Flags(),
			Action: start.Run(),
		}},
	}

	if err := app.Run(os.Args); err != nil {
		log.New().Fatal(err)
	}
}
