package system

import (
	"context"
	"runtime"
	"time"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

var (
	recv = make(chan segment, 1)
	send = make(chan segment, 1)
	poll = make(chan struct{}, 1)
	done = make(chan struct{})
)

func init() {
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-poll:
				runtime.Gosched()
			case <-ticker.C:
				// __poll()  // TODO:  activate?
			case <-done:
				return
			}
		}
	}()
}

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
	select {
	case <-done:
	default:
		close(done)
	}

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

func (socket) RecvMessage() (transport.IncomingMessage, error) {
	select {
	case seg := <-recv:
		defer free(seg)

		msg, err := capnp.Unmarshal(exports[seg])
		if err != nil {
			return nil, err
		}
		message, err := rpccp.ReadRootMessage(msg)
		if err != nil {
			return nil, err
		}
		return incoming(message), nil

	case <-done:
		return nil, rpc.ErrConnClosed
	}
}

// TODO:  export so that the host can trigger a poll on write
func __poll() {
	select {
	case poll <- struct{}{}:
	default:
	}
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

	for {
		select {
		case send <- export(b):
			__poll()
			return nil
		case <-done:
			return rpc.ErrConnClosed
		default:
			__poll()
		}
	}
}
