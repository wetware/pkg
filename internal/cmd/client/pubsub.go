package client

import (
	"fmt"
	"io"

	"github.com/urfave/cli/v2"
)

func Publish() *cli.Command {
	return &cli.Command{
		Name:    "publish",
		Aliases: []string{"pub"},
		Usage:   "publish a message from stdin to a pubsub topic",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "topic",
				Aliases:  []string{"t"},
				Usage:    "pubsub topic",
				Required: true,
			},
		},
		Before: dial(),
		Action: publish(),
	}
}

func Subscribe() *cli.Command {
	return &cli.Command{
		Name:    "subscribe",
		Aliases: []string{"sub"},
		Usage:   "print messages from a topic",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "topic",
				Aliases:  []string{"t"},
				Usage:    "pubsub topic",
				Required: true,
			},
		},
		Before: dial(),
		Action: subscribe(),
	}
}

func publish() cli.ActionFunc {
	return func(c *cli.Context) error {
		t := node.Join(c.Context, c.String("topic"))
		defer t.Release()

		b, err := io.ReadAll(c.App.Reader)
		if err != nil {
			return err
		}

		return t.Publish(c.Context, b)
	}
}

func subscribe() cli.ActionFunc {
	return func(c *cli.Context) error {
		t := node.Join(c.Context, c.String("topic"))
		defer t.Release()

		sub, err := t.Subscribe(c.Context)
		if err != nil {
			return err
		}
		defer sub.Cancel()

		for msg := range sub.C {
			fmt.Fprintln(c.App.Writer, string(msg))
		}

		return nil
	}
}
