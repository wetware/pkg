package client

import (
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
)

var logger = struct{ log.Logger }{log.New()}

func SetLogger(log log.Logger) { logger.Logger = log }

func Command() *cli.Command {
	return &cli.Command{
		Name:        "client",
		Usage:       "cli client for wetware clusters",
		Subcommands: commands,
	}
}

var commands = []*cli.Command{
	Discover(),
}

// ww client discover
func Discover() *cli.Command {
	return &cli.Command{
		Name:  "discover",
		Usage: "bootstrap client",
		Subcommands: []*cli.Command{
			Crawl(),
			Publish(),
		},
	}
}
