package client

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/wetware/ww/pkg/routing"
)

func subscribe(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:    "subscribe",
		Aliases: []string{"sub"},
		Flags:   subFlags(),
		Action:  subAction(ctx),
	}
}

func subFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "topic",
			Aliases: []string{"t"},
			Usage:   "pubsub topic",
		},
	}
}

func subAction(ctx context.Context) cli.ActionFunc {
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

		w := messagePrinter{
			topic: c.String("topic"),
			enc:   jsonEncoder(c.App.Writer, c.Bool("prettyprint")),
		}

		for msg := range sub.C {
			if err = w.PrintMessage(msg); err != nil {
				break
			}
		}

		return err
	}
}

func jsonEncoder(w io.Writer, pretty bool) (enc *json.Encoder) {
	if enc = json.NewEncoder(w); pretty {
		enc.SetIndent("", "  ")
	}

	return
}

type messagePrinter struct {
	topic string
	enc   *json.Encoder
}

func (m messagePrinter) PrintMessage(msg *pubsub.Message) error {
	if m.topic == "" {
		hb, err := routing.UnmarshalHeartbeat(msg.Data)
		if err != nil {
			return err
		}

		return m.enc.Encode(struct {
			Seq uint64        `json:"seq"`
			ID  peer.ID       `json:"id"`
			TTL time.Duration `json:"ttl"`
		}{
			Seq: binary.BigEndian.Uint64(msg.Seqno),
			ID:  msg.GetFrom(),
			TTL: hb.TTL(),
		})
	}

	// TODO(enhancement):  support s-exprs (or EDN) using github.com/polydawn/refmt
	if err := m.enc.Encode(msg.GetData); err != nil {
		logger.
			WithField("topic", m.topic).
			WithField("raw", string(msg.GetData())).
			Warn("failed to render message (currently, only JSON is supported)")
		return err
	}

	return nil
}
