package system

import (
	"context"
	"errors"
	"io"
	"runtime"
	"time"
	"unsafe"

	"github.com/stealthrocket/wazergo/types"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"capnproto.org/go/capnp/v3/rpc/transport"
	rpccp "capnproto.org/go/capnp/v3/std/capnp/rpc"
)

var (
	incoming = make(chan segment, 1)
	exports  = make(map[segment][]byte)
)

type segment struct {
	offset, length uint32
}

//go:inline
func bytesToPointer(b []byte) uint32 {
	return (*(*uint32)(unsafe.Pointer(unsafe.SliceData(b))))
}

// //go:inline
// func stringToPointer(s string) uint32 {
// 	return (*(*uint32)(unsafe.Pointer(unsafe.StringData(s))))
// }

func Login(ctx context.Context) (auth.Session, error) {
	// conn, err := FDSockDialer{}.DialRPC(ctx)
	// if err != nil {
	// 	return auth.Session{}, err
	// }
	// runtime.SetFinalizer(conn, func(c io.Closer) error {
	// 	return c.Close()
	// })
	conn := rpc.NewConn(make(guestTransport), nil)
	runtime.SetFinalizer(conn, func(c io.Closer) error {
		defer slog.Debug("called finalizer")
		return c.Close()
	})

	client := conn.Bootstrap(ctx)
	if err := client.Resolve(ctx); err != nil {
		return auth.Session{}, err
	}
	term := core.Terminal(client)

	f, release := term.Login(ctx, nil)
	defer release()

	res, err := f.Struct()
	if err != nil {
		return auth.Session{}, err
	}

	sess, err := res.Session()
	if err != nil {
		return auth.Session{}, err
	}

	return auth.Session(sess).AddRef(), nil
}

type guestTransport chan struct{}

func (guestTransport) NewMessage() (transport.OutgoingMessage, error) {
	msg, seg := capnp.NewMultiSegmentMessage(nil)
	body, err := rpccp.NewRootMessage(seg)
	if err != nil {
		return nil, err
	}

	return &message{
		body: body,
		send: func() error {
			b, err := msg.Marshal()
			if err != nil {
				return err
			}

			errno := sockSend(bytesToPointer(b), uint32(len(b)))
			if errno == 0 {
				return nil
			}

			return types.Errno(errno)
		},
	}, nil
}

func (closed guestTransport) RecvMessage() (transport.IncomingMessage, error) {
	select {
	case <-closed:
		return nil, errors.New("closed")

	case seg := <-incoming:
		defer delete(exports, seg) // unexport

		msg, err := capnp.Unmarshal(exports[seg])
		if err != nil {
			return nil, err
		}

		body, err := rpccp.ReadRootMessage(msg)
		if err != nil {
			return nil, err
		}

		return &message{body: body}, nil
	}
}

func (closed guestTransport) Close() error {
	close(closed)
	sockClose()
	return nil
}

type message struct {
	body rpccp.Message
	send func() error
}

func (m *message) Send() (err error) {
	// busy-loop until we're able to send
	// TODO:  lots of room for improvement here.
	for err = m.send(); err != nil; err = m.send() {
		time.Sleep(time.Millisecond)
	}

	m.send = nil
	return err
}

func (m message) Release() {
	m.body.Release()
}

func (m message) Message() rpccp.Message {
	return m.body
}
