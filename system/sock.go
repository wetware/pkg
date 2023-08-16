package system

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3/rpc/transport"
	"golang.org/x/exp/slog"
)

type Socket struct {
}

func (s Socket) Close() error {
	return nil
}

func (s Socket) Send(ctx context.Context, buf []byte) error {
	slog.Info("guest sent buffer to host",
		"bytes", len(buf))
	return nil
}

func (s Socket) Recv(ctx context.Context) ([]byte, error) {
	slog.Info("guest read buffer from host",
		"error", errors.New("NOT IMPLEMENTED"))

	return nil, errors.New("NOT IMPLEMENTED")
}

func (s Socket) NewMessage() (transport.OutgoingMessage, error) {
	<-context.Background().Done()
	return nil, errors.New("NOT IMPLEMENTED")
}

func (s Socket) RecvMessage() (transport.IncomingMessage, error) {
	<-context.Background().Done()
	return nil, errors.New("NOT IMPLEMENTED")
}
