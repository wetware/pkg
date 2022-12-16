package pubsub

import (
	"context"

	casm "github.com/wetware/casm/pkg"
	chan_api "github.com/wetware/ww/internal/api/channel"
	api "github.com/wetware/ww/internal/api/pubsub"
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

type handler chan []byte

func (ch handler) Params(ps api.Topic_subscribe_Params) error {
	ps.SetBuf(uint16(cap(ch)))
	return ps.SetChan(chan_api.Sender_ServerToClient(ch))
}

func (ch handler) Shutdown() { close(ch) }

func (ch handler) Next() (b []byte, ok bool) {
	b, ok = <-ch
	return
}

func (ch handler) Send(ctx context.Context, call chan_api.Sender_send) error {
	ptr, err := call.Args().Value()
	if err != nil {
		return err
	}

	// Copy the message data.  The segment will be zeroed when Send returns.
	msg := make([]byte, len(ptr.Data()))
	copy(msg, ptr.Data())

	// It's okay to block here, since there is only one writer.
	// Back-pressure will be handled by the BBR flow-limiter.
	select {
	case ch <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
