package system

import (
	"errors"

	"capnproto.org/go/capnp/v3/rpc/transport"
)

type socket struct{}

func (socket) Close() error {
	return nil
}

func (socket) NewMessage() (transport.OutgoingMessage, error) {
	return nil, errors.New("[ GUEST ]: NewMessage(): NOT IMPLEMENTED")
}

func (socket) RecvMessage() (transport.IncomingMessage, error) {
	return nil, errors.New("[ GUEST ]: RecvMessage(): NOT IMPLEMENTED")
}
