package client

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/lthibault/wetware/pkg/routing"
)

var sharedFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "topic",
		Aliases: []string{"t"},
		Usage:   "pubsub topic",
	},
}

func subFlags() []cli.Flag {
	return append(sharedFlags, []cli.Flag{
		&cli.BoolFlag{
			Name:    "prettyprint",
			Aliases: []string{"pretty", "pp"},
			Usage:   "indent JSON output",
		},
	}...)
}

func subAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		t, err := root.Join(c.String("topic"))
		if err != nil {
			return err
		}
		defer t.Close()

		sub, err := t.Subscribe(ctx)
		if err != nil {
			return err
		}
		logger.WithField("topic", t.String()).Debug("subscribed to topic")

		w := newMessagePrinter(c)
		for msg := range sub.C {
			if err = w.PrintMessage(msg); err != nil {
				break
			}
		}

		return err
	}
}

func pubFlags() []cli.Flag {
	return append(sharedFlags, []cli.Flag{
		// ...
	}...)
}

func pubAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		return errors.New("NOT IMPLEMENTED")
	}
}

type messagePrinter struct {
	topic string
	enc   *json.Encoder
}

func newMessagePrinter(c *cli.Context) messagePrinter {
	enc := json.NewEncoder(c.App.Writer)
	if c.Bool("prettyprint") {
		enc.SetIndent("", "  ")
	}

	return messagePrinter{

		topic: c.String("topic"),
		enc:   enc,
	}
}

func (m messagePrinter) PrintMessage(msg *pubsub.Message) error {
	if m.topic == "" {
		hb, err := routing.UnmarshalHeartbeat(msg.GetData())
		if err != nil {
			return err
		}

		return m.enc.Encode(struct {
			Seq uint64        `json:"seq"`
			ID  peer.ID       `json:"id"`
			TTL time.Duration `json:"ttl"`
		}{
			Seq: binary.BigEndian.Uint64(msg.Seqno),
			ID:  hb.ID(),
			TTL: hb.TTL(),
		})
	}

	if err := m.enc.Encode(msg.GetData); err != nil {
		logger.
			WithField("raw", string(msg.GetData())).
			Warn("failed to render message (currently, only JSON is supported)")
		return err
	}

	return nil
}
