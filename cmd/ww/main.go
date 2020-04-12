package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/lthibault/wetware/internal/cmd/start"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{{
			Name:   "start",
			Usage:  "start a host process",
			Action: start.Run(),
		}},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
