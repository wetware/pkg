package system

import (
	"context"
	"errors"
	"fmt"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
	"golang.org/x/sync/errgroup"
)

type Transport struct {
	Conn           *Conn
	tasks          *errgroup.Group
	cancel         context.CancelFunc
	recvCh, sendCh chan *capnp.Message
	closing        chan struct{}
}

func NewTransport(ctx context.Context, conn *Conn) Transport {
	ctx, cancel := context.WithCancel(ctx)
	tasks, ctx := errgroup.WithContext(ctx)

	t := Transport{
		Conn:    conn,
		tasks:   tasks,
		cancel:  cancel,
		recvCh:  make(chan *capnp.Message, 16),
		sendCh:  make(chan *capnp.Message, 16),
		closing: make(chan struct{}),
	}
	tasks.Go(t.recvLoop(ctx, conn))
	tasks.Go(t.sendLoop(ctx, conn))

	return t
}

func (t Transport) Close() error {
	defer close(t.closing)
	t.cancel()
	return t.tasks.Wait()
}

func (t Transport) NewMessage() (transport.OutgoingMessage, error) {
	m, seg := capnp.NewMultiSegmentMessage(nil)
	msg, err := rpccp.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	send := func() error {
		select {
		case t.sendCh <- m:
			return nil
		case <-t.closing:
			return errors.New("closing")
		}
	}

	return &outgoing{
		incoming: incoming(msg),
		send:     send,
	}, nil
}

func (t Transport) RecvMessage() (transport.IncomingMessage, error) {
	if m, ok := <-t.recvCh; ok {
		msg, err := rpccp.ReadRootMessage(m)
		return incoming(msg), err
	}

	return nil, errors.New("closed")
}

func (t Transport) recvLoop(ctx context.Context, conn *Conn) func() error {
	return func() error {
		defer close(t.recvCh)

		for buf := range conn.recvCh {
			m, err := capnp.Unmarshal(buf)
			if err != nil {
				return fmt.Errorf("umarshal capnp: %w", err)
			}

			select {
			case t.recvCh <- m:
			case <-ctx.Done():
				return nil
			}
		}

		return nil
	}
}

func (t Transport) sendLoop(ctx context.Context, conn *Conn) func() error {
	return func() error {
		defer close(conn.sendCh) // signal to conn.sendLoop that we are done

		for m := range t.sendCh {
			b, err := m.Marshal()
			if err != nil {
				return fmt.Errorf("marshal capnp: %w", err)
			}

			select {
			case conn.sendCh <- b:
				m.Release()

			case <-ctx.Done():
				return nil
			}
		}

		return nil
	}
}

type outgoing struct {
	incoming
	send func() error
}

func (out *outgoing) Send() error {
	err := out.send()
	out.send = nil
	return err
}

type incoming rpccp.Message

func (in incoming) Message() rpccp.Message {
	return rpccp.Message(in)
}

func (in incoming) Release() {
	in.Message().Release()
}
