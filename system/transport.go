package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
	"zenhack.net/go/util/rc"
)

type hostTransport struct {
	Sock *Socket
}

func (t hostTransport) Close() error {
	return t.Sock.Close(context.TODO())
}

func (t hostTransport) NewMessage() (transport.OutgoingMessage, error) {
	msg, seg := capnp.NewMultiSegmentMessage(nil)
	body, err := rpccp.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	ref := rc.NewRef(body, msg.Release)

	return &messageRef{
		body: ref,
		send: func() error {
			return t.Sock.Guest.Push(ref.AddRef())
		},
	}, nil
}

func (t hostTransport) RecvMessage() (transport.IncomingMessage, error) {
	ref, err := t.Sock.Host.Pop()
	if err != nil {
		return nil, err
	}

	return &messageRef{body: ref}, nil
}

type messageRef struct {
	body *rc.Ref[rpccp.Message]
	send func() error
}

func (m messageRef) Release() {
	m.body.Release()
}

func (m *messageRef) Send() error {
	err := m.send()
	m.send = nil
	return err
}

func (m messageRef) Message() rpccp.Message {
	return *m.body.Value()
}
