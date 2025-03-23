package server

import "github.com/urfave/cli/v2"

const Name = "server"

var DefaultCommand = "start"

func Commands() []*cli.Command {
	return []*cli.Command{
		Command(),
	}
}

func Command() *cli.Command {
	return &cli.Command{
		Name:  Name,
		Flags: []cli.Flag{
			// ...
		},
		Action: Start(),
	}
}

func Start() cli.ActionFunc {
	return func(c *cli.Context) error {
		return nil // TODO: implement server start
	}
}
