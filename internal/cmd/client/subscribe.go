package client

import (
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/urfave/cli/v2"
)

func subscribe(c *cli.Context) (err error) {
	var sub *pubsub.Subscription
	if topic := c.String("topic"); topic == "" {
		sub, err = node.GetClusterSubscription()
	} else {
		sub, err = node.PubSub().Subscribe(topic)
	}

	if err != nil {
		return
	}
	defer sub.Cancel()

	for {
		msg, err := sub.Next(c.Context)
		if err != nil {
			return err
		}

		fmt.Fprintln(c.App.Writer, string(msg.GetData()))
	}
}
