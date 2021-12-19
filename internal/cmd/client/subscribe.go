package client

import (
	"encoding/json"
	"fmt"
	"io"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/cluster/pulse"
)

type printer interface {
	Print(*pubsub.Message)
}

func subscribe(c *cli.Context) (err error) {
	var (
		sub *pubsub.Subscription
		p   printer
	)

	if topic := c.String("topic"); topic == "" {
		p = newHeartbeatPrinter(c.App.Writer)
		sub, err = node.GetClusterSubscription()
	} else {
		p = stringPrinter{c.App.Writer}
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

		p.Print(msg)
	}
}

type stringPrinter struct{ io.Writer }

func (p stringPrinter) Print(m *pubsub.Message) {
	fmt.Fprintln(p, string(m.Data))
}

type heartbeatPrinter struct {
	*json.Encoder
	hb pulse.Heartbeat
}

func newHeartbeatPrinter(w io.Writer) heartbeatPrinter {
	hb, err := pulse.NewHeartbeat(capnp.SingleSegment(nil))
	if err != nil {
		panic(err)
	}

	return heartbeatPrinter{
		Encoder: json.NewEncoder(w),
		hb:      hb,
	}
}

func (p heartbeatPrinter) Print(m *pubsub.Message) {
	if err := p.hb.UnmarshalBinary(m.Data); err != nil {
		logger.WithError(err).Error("failed to decode heartbeat")
	}

	p.Encoder.SetIndent("", "  ")
	p.Encode(struct {
		From peer.ID
		TTL  string
	}{
		From: peer.ID(m.From),
		TTL:  fmt.Sprintf("%v", p.hb.TTL()),
	})
}
