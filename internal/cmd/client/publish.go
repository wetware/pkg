package client

import (
	"context"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func publish(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "publish",
		Aliases: []string{"pub"},
		Flags:   pubFlags(),
		Action:  pubAction(ctx),
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

func pubAction(ctx context.Context) cli.ActionFunc {
	return func(c *cli.Context) error {
		return errors.New("NOT IMPLEMENTED")
	}
}
