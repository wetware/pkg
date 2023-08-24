package system

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
	"github.com/stealthrocket/wazergo/types"
)

type socket struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (sock socket) Close() error {
	sock.cancel()
	return nil
}

func (socket) NewMessage() (transport.OutgoingMessage, error) {
	// Alloc a local Message.  The send function will atomically:
	//   (1) Add (offset, size) tuple to the global export table
	//   (2) Make host call to add to queue in system.Socket{} (host side)
	_, seg := capnp.NewMultiSegmentMessage(nil)
	message, err := rpccp.NewRootMessage(seg)
	return outgoing(message), err
}

func (sock socket) RecvMessage() (transport.IncomingMessage, error) {
	buf, err := input.Recv(sock.ctx)
	if err != nil {
		return nil, err
	}

	msg, err := capnp.Unmarshal(buf)
	if err != nil {
		defer msg.Release()
		return nil, err
	}

	message, err := rpccp.ReadRootMessage(msg)
	if err != nil {
		defer msg.Release()
		return nil, err
	}

	return incoming(message), nil
}

type incoming rpccp.Message

func (msg incoming) Message() rpccp.Message {
	return rpccp.Message(msg)
}

func (msg incoming) Release() {
	capnp.Struct(msg).Message().Release()
}

type outgoing rpccp.Message

func (msg outgoing) Message() rpccp.Message {
	return incoming(msg).Message()
}

func (msg outgoing) Release() {
	incoming(msg).Release()
}

func (msg outgoing) Send() error {
	b, err := capnp.Struct(msg).Message().Marshal()
	if err != nil {
		return err
	}

	seg := export(b)
	if status := send(seg.offset, seg.length); status != 0 {
		return types.Errno(status)
	}

	return nil
}
