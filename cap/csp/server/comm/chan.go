package comm

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/channel"
	"github.com/wetware/pkg/util/casm"
)

var (
	ErrEmpty  = errors.New("empty")
	ErrClosed = errors.New("closed")
)

type ( // server methods
	MethodClose         = api.Closer_close
	MethodSend          = api.Sender_send
	MethodRecv          = api.Recver_recv
	MethodNewSender     = api.SendCloser_newSender
	MethodNewCloser     = api.SendCloser_newCloser
	MethodNewRecver     = api.Chan_newRecver
	MethodNewSendCloser = api.Chan_newSendCloser
)

type ( // server interfaces
	CloseServer interface {
		Close(context.Context, MethodClose) error
	}

	SendServer interface {
		Send(context.Context, MethodSend) error
	}

	RecvServer interface {
		Recv(context.Context, MethodRecv) error
	}

	SendCloseServer interface {
		SendServer
		CloseServer
		NewSender(context.Context, MethodNewSender) error
		NewCloser(context.Context, MethodNewCloser) error
	}

	ChanServer interface {
		RecvServer
		SendCloseServer
		NewRecver(context.Context, MethodNewRecver) error
		NewSendCloser(context.Context, MethodNewSendCloser) error
	}
)

type Chan api.Chan

func NewChan(s ChanServer) Chan {
	return Chan(api.Chan_ServerToClient(s))
}

func (c Chan) Client() capnp.Client {
	return capnp.Client(c)
}

func (c Chan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func (c Chan) Send(ctx context.Context, v Value) error {
	return Sender(c).Send(ctx, v)
}

func (c Chan) Recv(ctx context.Context) (casm.Future, capnp.ReleaseFunc) {
	return Recver(c).Recv(ctx)
}

func (c Chan) AddRef() Chan {
	return Chan(c.Client().AddRef())
}

func (c Chan) Release() {
	c.Client().Release()
}

func (c Chan) NewCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	return SendCloser(c).NewCloser(ctx)
}

func (c Chan) NewSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	return SendCloser(c).NewSender(ctx)
}

func (c Chan) NewRecver(ctx context.Context) (Recver, capnp.ReleaseFunc) {
	f, release := api.Chan(c).NewRecver(ctx, nil)
	return Recver(f.Recver()), release
}

func (c Chan) NewSendCloser(ctx context.Context) (SendCloser, capnp.ReleaseFunc) {
	f, release := api.Chan(c).NewSendCloser(ctx, nil)
	return SendCloser(f.SendCloser()), release
}

type Sender api.Sender

func NewSender(s SendServer) Sender {
	return Sender(api.Sender_ServerToClient(s))
}

func (s Sender) Client() capnp.Client {
	return capnp.Client(s)
}

func (s Sender) Send(ctx context.Context, v Value) error {
	f, release := api.Sender(s).Send(ctx, v)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (s Sender) AddRef() Sender {
	return Sender(s.Client().AddRef())
}

func (s Sender) Release() {
	s.Client().Release()
}

type Recver api.Recver

func NewRecver(r RecvServer) Recver {
	return Recver(api.Recver_ServerToClient(r))
}

func (r Recver) Client() capnp.Client {
	return capnp.Client(r)
}

func (r Recver) Recv(ctx context.Context) (casm.Future, capnp.ReleaseFunc) {
	f, release := api.Recver(r).Recv(ctx, nil)
	return casm.Future{Future: f.Value()}, release
}

func (r Recver) AddRef() Recver {
	return Recver(r.Client().AddRef())
}

func (r Recver) Release() {
	r.Client().Release()
}

type SendCloser api.SendCloser

func NewSendCloser(sc SendCloseServer) SendCloser {
	return SendCloser(api.SendCloser_ServerToClient(sc))
}

func (sc SendCloser) Client() capnp.Client {
	return capnp.Client(sc)
}

func (sc SendCloser) Close(ctx context.Context) error {
	return Closer(sc).Close(ctx)
}

func (sc SendCloser) Send(ctx context.Context, v Value) error {
	return Sender(sc).Send(ctx, v)
}

func (sc SendCloser) AddRef() SendCloser {
	return SendCloser(sc.Client().AddRef())
}

func (sc SendCloser) Release() {
	sc.Client().Release()
}

func (sc SendCloser) NewSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).NewSender(ctx, nil)
	return Sender(f.Sender()), release
}

func (sc SendCloser) NewCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).NewCloser(ctx, nil)
	return Closer(f.Closer()), release
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

// Client takes any client-like type and converts it into a value
// capable of being sent through a channel.
func Client[T ~capnp.ClientKind](t T) Value {
	return func(ps api.Sender_send_Params) error {
		id := ps.Message().CapTable().Add(capnp.Client(t))
		ifc := capnp.NewInterface(ps.Segment(), id)
		return ps.SetValue(ifc.ToPtr())
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

// // Future result from a Chan operation. It is a specialized instance
// // of a casm.Future that provides typed methods for common capnp.Ptr
// // types.
// type Future struct{ casm.Future }

// // Value returns a *Future that asynchronously resolves to the  next
// // value produced by the channel.
// func (f Future) Value() *capnp.Future {
// 	return f.Field(0, nil)
// }

// func (f Future) Client() capnp.Client {
// 	<-f.Done() // avoids returning *promised* client
// 	return f.Value().Client()
// }

// func (f Future) Ptr() (capnp.Ptr, error) {
// 	return f.Value().Ptr()
// }

// func (f Future) Struct() (capnp.Struct, error) {
// 	ptr, err := f.Ptr()
// 	return ptr.Struct(), err
// }

// func (f Future) List() (capnp.List, error) {
// 	ptr, err := f.Ptr()
// 	return ptr.List(), err
// }

// func (f Future) Bytes() ([]byte, error) {
// 	ptr, err := f.Ptr()
// 	return ptr.Data(), err
// }

// func (f Future) Text() (string, error) {
// 	ptr, err := f.Ptr()
// 	return ptr.Text(), err
// }
