package system

import (
	"context"
	"errors"
	"log/slog"
	"runtime"

	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
)

type Socket struct{}

func (sock *Socket) Close() error {
	return nil
}

func (sock *Socket) Poll(ctx context.Context) func() {
	slog.Info("Guest called runtime.Gosched()!") // and we picked it up on the host side!

	return runtime.Gosched
}

func (sock *Socket) Transport() rpc.Transport {
	return capnpTransport{sock: sock}
}

type capnpTransport struct {
	sock *Socket
}

func (t capnpTransport) Close() error {
	return t.sock.Close()
}

func (t capnpTransport) NewMessage() (transport.OutgoingMessage, error) {
	return nil, errors.New("NewMessage(): NOT IMPLEMENTED")
}

func (t capnpTransport) RecvMessage() (transport.IncomingMessage, error) {
	return nil, errors.New("RecvMessage(): NOT IMPLEMENTED")
}
