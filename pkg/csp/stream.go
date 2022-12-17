package csp

import (
	"context"

	"capnproto.org/go/capnp/v3/exp/clock"
	"capnproto.org/go/capnp/v3/flowcontrol/bbr"
	"github.com/wetware/casm/pkg/util/stream"
	"github.com/wetware/ww/internal/api/channel"
)

type SendStream struct {
	ctx    context.Context
	stream *stream.Stream[channel.Sender_send_Params]
}

// NewStream for the sender.   This will overwrite the existing
// flow limiter. Callers SHOULD NOT create more than one stream
// for a given sender.
func (s Sender) NewStream(ctx context.Context) SendStream {
	sender := channel.Sender(s)
	sender.SetFlowLimiter(bbr.NewLimiter(clock.System))

	return SendStream{
		ctx:    ctx,
		stream: stream.New(channel.Sender(s).Send),
	}
}

func (s SendStream) Send(v Value) (err error) {
	if s.stream.Call(s.ctx, v); !s.stream.Open() {
		err = s.Close()
	}

	return
}

func (s SendStream) Close() error {
	return s.stream.Wait()
}
