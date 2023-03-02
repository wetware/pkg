package csp

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/channel"
)

var (
	ErrEmpty  = errors.New("empty")
	ErrClosed = errors.New("closed")
)

type ( // server methods
	MethodClose        = api.Closer_close
	MethodSend         = api.Sender_send
	MethodRecv         = api.Recver_recv
	MethodAsSender     = api.SendCloser_asSender
	MethodAsCloser     = api.SendCloser_asCloser
	MethodAsRecver     = api.Chan_asRecver
	MethodAsSendCloser = api.Chan_asSendCloser
)

type ( // server interfaces
	CloseServer interface {
		Close(context.Context, MethodClose) error
	}

	SendServer interface {
		Cap() uint
		Send(context.Context, MethodSend) error
	}

	RecvServer interface {
		Cap() uint
		Recv(context.Context, MethodRecv) error
	}

	SendCloseServer interface {
		SendServer
		CloseServer
		AsSender(context.Context, MethodAsSender) error
		AsCloser(context.Context, MethodAsCloser) error
	}

	ChanServer interface {
		RecvServer
		SendCloseServer
		AsRecver(context.Context, MethodAsRecver) error
		AsSendCloser(context.Context, MethodAsSendCloser) error
	}
)

/*
	Client interfaces
*/

type Chan interface {
	Client() capnp.Client

	AddRef() Chan
	Release()

	Send(context.Context, Value) error
	Recv(context.Context) (Future, capnp.ReleaseFunc)
	Close(context.Context) error

	AsSender(context.Context) (Sender, capnp.ReleaseFunc)
	AsRecver(context.Context) (Recver, capnp.ReleaseFunc)
	AsCloser(context.Context) (Closer, capnp.ReleaseFunc)
	AsSendCloser(context.Context) (SendCloser, capnp.ReleaseFunc)
}

func NewChan(s ChanServer) Chan {
	if s.Cap() == 0 {
		return SyncChan(api.Chan_ServerToClient(s))
	}

	panic("AsyncServer NOT IMPLEMENTED")
}

type Sender interface {
	Client() capnp.Client
	Send(context.Context, Value) error
	AddRef() Sender
	Release()
}

func NewSender(s SendServer) SyncSender {
	switch ss := s.(type) {
	case *SyncServer:
		return SyncSender(api.Sender_ServerToClient(s))
	// case *AsyncServer:
	// 	return AsyncChan(api.Chan_ServerToClient(s))

	default:
		panic(fmt.Sprintf("unrecognized chan type: %s", reflect.TypeOf(ss)))
	}
}

type Recver interface {
	Client() capnp.Client
	Recv(context.Context) (Future, capnp.ReleaseFunc)
	AddRef() Recver
	Release()
}

func NewRecver(r RecvServer) SyncRecver {
	switch rs := r.(type) {
	case *SyncServer:
		return SyncRecver(api.Recver_ServerToClient(r))
	// case *AsyncServer:
	// 	return AsyncChan(api.Chan_ServerToClient(s))

	default:
		panic(fmt.Sprintf("unrecognized chan type: %s", reflect.TypeOf(rs)))
	}
}

type SendCloser interface {
	AddRef() SendCloser
	Release()

	Send(context.Context, Value) error
	Close(context.Context) error

	AsSender(context.Context) (Sender, capnp.ReleaseFunc)
	AsCloser(context.Context) (Closer, capnp.ReleaseFunc)
}

func NewSendCloser(sc SendCloseServer) SyncSendCloser {
	switch s := sc.(type) {
	case *SyncServer:
		return SyncSendCloser(api.SendCloser_ServerToClient(sc))
	// case *AsyncServer:
	// 	return AsyncChan(api.Chan_ServerToClient(s))

	default:
		panic(fmt.Sprintf("unrecognized chan type: %s", reflect.TypeOf(s)))
	}
}

type Closer api.Chan

func NewCloser(c CloseServer) Closer {
	return Closer(api.Closer_ServerToClient(c))
}

func (c Closer) Close(ctx context.Context) error {
	f, release := api.Closer(c).Close(ctx, nil)
	defer release()

	_, err := f.Struct()
	return err
}

func (c Closer) AddRef() Closer {
	return Closer(capnp.Client(c).AddRef())
}

func (c Closer) Release() {
	capnp.Client(c).Release()
}

type Value func(api.Sender_send_Params) error

// Ptr takes any capnp pointer and converts it into a value
// capable of being sent through a channel.
func Ptr(ptr capnp.Ptr) Value {
	return func(ps api.Sender_send_Params) error {
		return ps.SetValue(ptr)
	}
}

// Struct takes any capnp struct and converts it into a value
// capable of being sent through a channel.
func Struct[T ~capnp.StructKind](t T) Value {
	return Ptr(capnp.Struct(t).ToPtr())
}

// List takes any capnp list and converts it into a value capable
// of being sent through a channel.
func List[T ~capnp.ListKind](t T) Value {
	return Ptr(capnp.List(t).ToPtr())
}

// Data takes any []byte-like type and converts it into a value
// capable of being sent through a channel.
func Data[T ~[]byte](t T) Value {
	return func(ps api.Sender_send_Params) error {
		return capnp.Struct(ps).SetData(0, []byte(t))
	}
}

// Text takes any string-like type and converts it into a value
// capable of being sent through a channel.
func Text[T ~string](t T) Value {
	return func(ps api.Sender_send_Params) error {
		return capnp.Struct(ps).SetText(0, string(t))
	}
}

// Future result from a Chan operation. It is a specialized instance
// of a casm.Future that provides typed methods for common capnp.Ptr
// types.
type Future casm.Future

func (f Future) Value() *capnp.Future {
	return f.Field(0, nil)
}

func (f Future) Client() capnp.Client {
	return f.Value().Client()
}

func (f Future) Ptr() (capnp.Ptr, error) {
	return f.Value().Ptr()
}

func (f Future) Struct() (capnp.Struct, error) {
	ptr, err := f.Ptr()
	return ptr.Struct(), err
}

func (f Future) List() (capnp.List, error) {
	ptr, err := f.Ptr()
	return ptr.List(), err
}

func (f Future) Bytes() ([]byte, error) {
	ptr, err := f.Ptr()
	return ptr.Data(), err
}

func (f Future) Text() (string, error) {
	ptr, err := f.Ptr()
	return ptr.Text(), err
}
