package client

import (
	"fmt"
	"io"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/client"
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

		// FIXME:  if we remove this, commands with sochastically hang.
		//         Best guess is that this happens when they are pipelined
		//         against a pending FutureTopic.
		time.Sleep(time.Millisecond * 5)

		return t.Publish(c.Context, b)
	}
}

func subscribe() cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		var (
			sub client.Subscription
			msg []byte
		)

		t := node.Join(c.Context, c.String("topic"))
		defer t.Release()

		sub, err = t.Subscribe(c.Context)
		if err != nil {
			return
		}
		defer sub.Cancel()

		for {
			msg, err = sub.Next(c.Context)
			if err != nil {
				break
			}

			fmt.Fprintln(c.App.Writer, string(msg))
		}

		if err == client.ErrDisconnected {
			err = nil
		}

		return err
	}
}
