//go:generate mockgen -source=chan.go -destination=../../internal/mock/pkg/csp/chan.go -package=mock_csp

package csp

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/flowcontrol"
	casm "github.com/wetware/casm/pkg"
	"github.com/wetware/casm/pkg/util/stream"
	"github.com/wetware/ww/internal/api/channel"
)

var (
	ErrEmpty  = errors.New("empty")
	ErrClosed = errors.New("closed")
)

type (
	MethodClose = channel.Closer_close
	MethodSend  = channel.Sender_send
	MethodRecv  = channel.Recver_recv
)

type CloseServer interface {
	Close(context.Context, MethodClose) error
}

type SendServer interface {
	Send(context.Context, MethodSend) error
}

type RecvServer interface {
	Recv(context.Context, MethodRecv) error
}

type SendCloseServer interface {
	SendServer
	CloseServer
}

type ChanServer interface {
	CloseServer
	SendServer
	RecvServer
}

type Value func(channel.Sender_send_Params) error

// Ptr takes any capnp pointer and converts it into a value
// capable of being sent through a channel.
func Ptr(ptr capnp.Ptr) Value {
	return func(ps channel.Sender_send_Params) error {
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
	return func(ps channel.Sender_send_Params) error {
		return capnp.Struct(ps).SetData(0, []byte(t))
	}
}

// Text takes any string-like type and converts it into a value
// capable of being sent through a channel.
func Text[T ~string](t T) Value {
	return func(ps channel.Sender_send_Params) error {
		return capnp.Struct(ps).SetText(0, string(t))
	}
}

// Future result from a Chan operation. It is a specialized instance
// of a casm.Future that provides typed methods for common capnp.Ptr
// types.
type Future casm.Future

func (f Future) Await(ctx context.Context) (val capnp.Ptr, err error) {
	if err = casm.Future(f).Await(ctx); err == nil {
		val, err = f.Ptr()
	}

	return
}

func (f Future) AwaitBytes(ctx context.Context) ([]byte, error) {
	ptr, err := f.Await(ctx)
	return ptr.Data(), err
}

func (f Future) AwaitString(ctx context.Context) (string, error) {
	ptr, err := f.Await(ctx)
	return ptr.Text(), err
}

func (f Future) Ptr() (capnp.Ptr, error) {
	res, err := channel.Recver_recv_Results_Future(f).Struct()
	if err != nil {
		return capnp.Ptr{}, err
	}

	return res.Value()
}

func (f Future) Client() capnp.Client {
	return f.Future.Field(0, nil).Client()
}

func (f Future) Struct() (capnp.Struct, error) {
	ptr, err := f.Ptr()
	return ptr.Struct(), err
}

func (f Future) Data() ([]byte, error) {
	ptr, err := f.Ptr()
	return ptr.Data(), err
}

func (f Future) Text() (string, error) {
	ptr, err := f.Ptr()
	return ptr.Text(), err
}

type Chan channel.Chan

func NewChan(s ChanServer) Chan {
	return Chan(channel.Chan_ServerToClient(s))
}

func (c Chan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func (c Chan) Send(ctx context.Context, v Value) (casm.Future, capnp.ReleaseFunc) {
	return Sender(c).Send(ctx, v)
}

func (c Chan) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	f, release := Recver(c).Recv(ctx)
	return Future(f), release
}

// NewStream for the sender.   This will overwrite the existing
// flow limiter. Callers SHOULD NOT create more than one stream
// for a given sender.
func (c Chan) NewStream(ctx context.Context) SendStream {
	return Sender(c).NewStream(ctx)
}

func (c Chan) AddRef() Chan {
	return Chan(capnp.Client(c).AddRef())
}

func (c Chan) Release() {
	capnp.Client(c).Release()
}

type SendCloser Chan

func NewSendCloser(sc SendCloseServer) SendCloser {
	return SendCloser(channel.SendCloser_ServerToClient(sc))
}

func (sc SendCloser) Close(ctx context.Context) error {
	return Closer(sc).Close(ctx)
}

func (sc SendCloser) Send(ctx context.Context, v Value) (casm.Future, capnp.ReleaseFunc) {
	return Sender(sc).Send(ctx, v)
}

// NewStream for the sender.   This will overwrite the existing
// flow limiter. Callers SHOULD NOT create more than one stream
// for a given sender.
func (sc SendCloser) NewStream(ctx context.Context) SendStream {
	return Sender(sc).NewStream(ctx)
}

func (sc SendCloser) AddRef() SendCloser {
	return SendCloser(capnp.Client(sc).AddRef())
}

func (sc SendCloser) Release() {
	capnp.Client(sc).Release()
}

type Sender Chan

func NewSender(s SendServer) Sender {
	return Sender(channel.Sender_ServerToClient(s))
}

func (s Sender) Send(ctx context.Context, v Value) (casm.Future, capnp.ReleaseFunc) {
	f, release := channel.Sender(s).Send(ctx, v)
	return casm.Future(f), release
}

// NewStream for the sender.   This will overwrite the existing
// flow limiter. Callers SHOULD NOT create more than one stream
// for a given sender.
func (s Sender) NewStream(ctx context.Context) SendStream {
	sender := channel.Sender(s)
	sender.SetFlowLimiter(flowcontrol.NewFixedLimiter(1e6)) // TODO:  use BBR once scheduler bug is fixed

	return SendStream{
		ctx:    ctx,
		stream: stream.New(sender.Send),
	}
}

func (s Sender) AddRef() Sender {
	return Sender(capnp.Client(s).AddRef())
}

func (s Sender) Release() {
	capnp.Client(s).Release()
}

type Recver Chan

func NewRecver(r RecvServer) Recver {
	return Recver(channel.Recver_ServerToClient(r))
}

func (r Recver) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	f, release := channel.Recver(r).Recv(ctx, nil)
	return Future(f), release
}

func (r Recver) AddRef() Recver {
	return Recver(capnp.Client(r).AddRef())
}

func (r Recver) Release() {
	capnp.Client(r).Release()
}

type Closer Chan

func NewCloser(c CloseServer) Closer {
	return Closer(channel.Closer_ServerToClient(c))
}

func (c Closer) Close(ctx context.Context) error {
	f, release := channel.Closer(c).Close(ctx, nil)
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
