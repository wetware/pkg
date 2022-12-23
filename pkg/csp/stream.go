package csp

import (
	"context"

	"github.com/wetware/casm/pkg/util/stream"
	"github.com/wetware/ww/internal/api/channel"
)

type SendStream struct {
	ctx    context.Context
	stream *stream.Stream[channel.Sender_send_Params]
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
