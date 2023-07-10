package pubsub

import (
	"context"

	"capnproto.org/go/capnp/v3/exp/bufferpool"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/api/pubsub"
)

// Subscription is a stateful iterator over a stream of topic messages.
type Subscription casm.Iterator[[]byte]

// Next blocks until the next message is received, and returns it.  It
// returns nil when the subscription is canceled.
func (sub Subscription) Next() []byte {
	b, _ := casm.Iterator[[]byte](sub).Next()
	return b
}

// Err returns the first non-nil error encountered by the subscription.
// If there is no error, Err() returns nil.
func (sub Subscription) Err() error {
	return casm.Iterator[[]byte](sub).Err()
}

type consumer chan []byte

func (ch consumer) Params(ps api.Topic_subscribe_Params) error {
	ps.SetBuf(uint16(cap(ch)))
	return ps.SetConsumer(api.Topic_Consumer_ServerToClient(ch))
}

func (ch consumer) Shutdown() { close(ch) }

func (ch consumer) Next() (b []byte, ok bool) {
	b, ok = <-ch
	return
}

func (ch consumer) Consume(ctx context.Context, call api.Topic_Consumer_consume) error {
	msg, err := call.Args().Msg()
	if err != nil {
		return err
	}

	// Copy the message data.  The segment will be zeroed when Send returns.
	buf := bufferpool.Default.Get(len(msg))
	copy(buf, msg)

	// It's okay to block here, since there is only one writer.
	// Back-pressure will be handled by the BBR flow-limiter.
	select {
	case ch <- buf:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
