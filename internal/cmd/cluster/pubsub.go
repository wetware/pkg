package cluster

import (
	"fmt"
	"io"

	"github.com/urfave/cli/v2"
)

func publish() *cli.Command {
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
		Before: setup(),
		After:  teardown(),
		Action: pubAction(),
	}
}

func subscribe() *cli.Command {
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
			&cli.BoolFlag{
				Name:  "hex",
				Usage: "format output as hex",
			},
		},
		Before: setup(),
		After:  teardown(),
		Action: subAction(),
	}
}

func pubAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		t, release := node.Join(c.Context, c.String("topic"))
		defer release()

		b, err := io.ReadAll(c.App.Reader)
		if err != nil {
			return err
		}

		return t.Publish(c.Context, b)
	}
}

func subAction() cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		t, release := node.Join(c.Context, c.String("topic"))
		defer release()

		sub, cancel := t.Subscribe(c.Context)
		defer cancel()

		print := newPrinter(c)

		for msg := sub.Next(); msg != nil; msg = sub.Next() {
			print(msg)
		}

		return sub.Err()
	}
}

func newPrinter(c *cli.Context) func([]byte) {
	var format = "%s\n"
	if c.Bool("hex") {
		format = "%x\n"
	}

	return func(b []byte) {
		fmt.Fprintf(c.App.Writer, format, b)
	}
}
