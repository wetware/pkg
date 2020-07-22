package client

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func publish() *cli.Command {
	return &cli.Command{
		Name:    "publish",
		Aliases: []string{"pub"},
		Flags:   pubFlags(),
		Action:  pubAction(),
	}
}

func pubFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "topic",
			Aliases: []string{"t"},
			Usage:   "pubsub topic",
		},
	}
}

func pubAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		return errors.New("NOT IMPLEMENTED")
	}
}
