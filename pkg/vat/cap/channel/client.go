package channel

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/ww/internal/api/channel"
	"github.com/wetware/ww/pkg/vat"
)

type Value func(channel.Sender_send_Params) error

func Ptr(ptr capnp.Ptr) Value {
	return func(ps channel.Sender_send_Params) error {
		return ps.SetValue(ptr)
	}
}

func Data(b []byte) Value {
	return func(ps channel.Sender_send_Params) error {
		return capnp.Struct(ps).SetData(0, b)
	}
}

func Text(s string) Value {
	return func(ps channel.Sender_send_Params) error {
		return capnp.Struct(ps).SetText(0, s)
	}
}

type Chan channel.Chan

func New(s Server) Chan {
	return Chan(channel.Chan_ServerToClient(s))
}

func (c Chan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func (c Chan) Send(ctx context.Context, v Value) (vat.Future, capnp.ReleaseFunc) {
	return Sender(c).Send(ctx, v)
}

func (c Chan) Recv(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	return Recver(c).Recv(ctx)
}

func (c Chan) Client() capnp.Client {
	return capnp.Client(c)
}

func (c Chan) AddRef() Chan {
	return Chan(c.Client().AddRef())
}

func (c Chan) Release() {
	c.Client().Release()
}

type PeekableChan channel.Chan

func (c PeekableChan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func NewPeekableChan(s PeekableServer) PeekableChan {
	return PeekableChan(channel.PeekableChan_ServerToClient(s))
}

func (c PeekableChan) Send(ctx context.Context, v Value) (vat.Future, capnp.ReleaseFunc) {
	return Sender(c).Send(ctx, v)
}

func (c PeekableChan) Recv(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	return Recver(c).Recv(ctx)
}

func (c PeekableChan) Client() capnp.Client {
	return capnp.Client(c)
}

func (c PeekableChan) AddRef() PeekableChan {
	return PeekableChan(c.Client().AddRef())
}

func (c PeekableChan) Release() {
	c.Client().Release()
}

type SendCloser Chan

func NewSendCloser(sc SendCloseServer) SendCloser {
	return SendCloser(channel.SendCloser_ServerToClient(sc))
}

func (sc SendCloser) Close(ctx context.Context) error {
	return Closer(sc).Close(ctx)
}

func (sc SendCloser) Send(ctx context.Context, v Value) (vat.Future, capnp.ReleaseFunc) {
	return Sender(sc).Send(ctx, v)
}

func (sc SendCloser) Client() capnp.Client {
	return capnp.Client(sc)
}

func (sc SendCloser) AddRef() SendCloser {
	return SendCloser(sc.Client().AddRef())
}

func (sc SendCloser) Release() {
	sc.Client().Release()
}

type PeekRecver Chan

func NewPeekRecver(pr PeekRecvServer) PeekRecver {
	return PeekRecver(channel.PeekRecver_ServerToClient(pr))
}

func (pr PeekRecver) Peek(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	return Peeker(pr).Peek(ctx)
}

func (pr PeekRecver) Recv(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	return Recver(pr).Recv(ctx)
}

func (pr PeekRecver) Client() capnp.Client {
	return capnp.Client(pr)
}

func (pr PeekRecver) AddRef() PeekRecver {
	return PeekRecver(pr.Client().AddRef())
}

func (pr PeekRecver) Release() {
	pr.Client().Release()
}

type Sender Chan

func NewSender(s SendServer) Sender {
	return Sender(channel.Sender_ServerToClient(s))
}

func (s Sender) Send(ctx context.Context, v Value) (vat.Future, capnp.ReleaseFunc) {
	f, release := channel.Sender(s).Send(ctx, v)
	return vat.Future(f), release
}

func (s Sender) Client() capnp.Client {
	return capnp.Client(s)
}

func (s Sender) AddRef() Sender {
	return Sender(s.Client().AddRef())
}

func (s Sender) Release() {
	s.Client().Release()
}

type Peeker Chan

func NewPeeker(p PeekServer) Peeker {
	return Peeker(channel.Peeker_ServerToClient(p))
}

func (p Peeker) Peek(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	f, release := channel.Peeker(p).Peek(ctx, nil)
	return vat.FuturePtr(f), release
}

func (p Peeker) Client() capnp.Client {
	return capnp.Client(p)
}

func (p Peeker) AddRef() Peeker {
	return Peeker(p.Client().AddRef())
}

func (p Peeker) Release() {
	p.Client().Release()
}

type Recver Chan

func NewRecver(r RecvServer) Recver {
	return Recver(channel.Recver_ServerToClient(r))
}

func (r Recver) Recv(ctx context.Context) (vat.FuturePtr, capnp.ReleaseFunc) {
	f, release := channel.Recver(r).Recv(ctx, nil)
	return vat.FuturePtr(f), release
}

func (r Recver) Client() capnp.Client {
	return capnp.Client(r)
}

func (r Recver) AddRef() Recver {
	return Recver(r.Client().AddRef())
}

func (r Recver) Release() {
	r.Client().Release()
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

func (c Closer) Client() capnp.Client {
	return capnp.Client(c)
}

func (c Closer) AddRef() Closer {
	return Closer(c.Client().AddRef())
}

func (c Closer) Release() {
	c.Client().Release()
}
