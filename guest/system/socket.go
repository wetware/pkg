package system

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

func Bootstrap[T ~capnp.ClientKind](ctx context.Context) (T, capnp.ReleaseFunc) {
	conn := rpc.NewConn(socket{}, nil)

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		defer conn.Close()
		return failure[T](err)
	}

	return T(client), func() {
		client.Release()
		conn.Close()
	}
}

func failure[T ~capnp.ClientKind](err error) (T, capnp.ReleaseFunc) {
	return T(capnp.ErrorClient(err)), func() {}
}

type socket struct{}

func (socket) Close() error {
	return nil
}

func (socket) NewMessage() (transport.OutgoingMessage, error) {
	defer runtime.Gosched() // Give the host an opportunity to consume the message

	// Alloc a local Message.  The send function will atomically:
	//   (1) Add (offset, size) tuple to the global export table
	//   (2) Make host call to add to queue in system.Socket{} (host side)
	_, seg := capnp.NewMultiSegmentMessage(nil)
	message, err := rpccp.NewRootMessage(seg)
	return outgoing(message), err

}

func (socket) RecvMessage() (transport.IncomingMessage, error) {
	for ; ; runtime.Gosched() {
		if message := poll(); message != nil {
			return read(message)
		}

		time.Sleep(time.Microsecond * 100)
	}
}

func poll() []byte {
	slog.Warn("stub call to guest/system.poll()")
	return nil
}

func read(message []byte) (transport.IncomingMessage, error) {
	return nil, errors.New("guest/system.read()::NOT IMPLEMENTED")
}

type outgoing rpccp.Message

func (msg outgoing) Message() rpccp.Message {
	return rpccp.Message(msg)
}

func (msg outgoing) Release() {
	capnp.Struct(msg).Message().Release()
}

func (msg outgoing) Send() error {
	// TODO:  marshal msg and add the bytes to the export table

	return errors.New("guest/system.outgoing.Send()::NOT IMPLEMENTED")
}
