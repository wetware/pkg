package csp

import (
	"context"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/channel"
)

type AsyncChan api.Chan

func (c AsyncChan) Client() capnp.Client {
	return capnp.Client(c)
}

func (c AsyncChan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func (c AsyncChan) Send(ctx context.Context, v Value) error {
	return AsyncSender(c).Send(ctx, v)
}

func (c AsyncChan) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	return AsyncRecver(c).Recv(ctx)
}

func (c AsyncChan) AddRef() Chan {
	return AsyncChan(c.Client().AddRef())
}

func (c AsyncChan) Release() {
	c.Client().Release()
}

func (c AsyncChan) AsCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	return AsyncSendCloser(c).AsCloser(ctx)
}

func (c AsyncChan) AsSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	return AsyncSendCloser(c).AsSender(ctx)
}

func (c AsyncChan) AsRecver(ctx context.Context) (Recver, capnp.ReleaseFunc) {
	f, release := api.Chan(c).AsRecver(ctx, nil)
	return AsyncRecver(f.Recver()), release
}

func (c AsyncChan) AsSendCloser(ctx context.Context) (SendCloser, capnp.ReleaseFunc) {
	f, release := api.Chan(c).AsSendCloser(ctx, nil)
	return AsyncSendCloser(f.SendCloser()), release
}

type AsyncSendCloser api.SendCloser

func (sc AsyncSendCloser) Client() capnp.Client {
	return capnp.Client(sc)
}

func (sc AsyncSendCloser) Close(ctx context.Context) error {
	return Closer(sc).Close(ctx)
}

func (sc AsyncSendCloser) Send(ctx context.Context, v Value) error {
	return AsyncSender(sc).Send(ctx, v)
}

func (sc AsyncSendCloser) AddRef() SendCloser {
	return AsyncSendCloser(sc.Client().AddRef())
}

func (sc AsyncSendCloser) Release() {
	sc.Client().Release()
}

func (sc AsyncSendCloser) AsSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).AsSender(ctx, nil)
	return AsyncSender(f.Sender()), release
}

func (sc AsyncSendCloser) AsCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).AsCloser(ctx, nil)
	return Closer(f.Closer()), release
}

type AsyncSender api.Sender

func (s AsyncSender) Client() capnp.Client {
	return capnp.Client(s)
}

func (s AsyncSender) Send(ctx context.Context, v Value) error {
	f, release := api.Sender(s).Send(ctx, func(ps api.Sender_send_Params) error {
		ps.SetAsync(true)
		return v(ps)
	})
	defer release()

	return casm.Future(f).Await(ctx)
}

func (s AsyncSender) AddRef() Sender {
	return AsyncSender(s.Client().AddRef())
}

func (s AsyncSender) Release() {
	s.Client().Release()
}

type AsyncRecver api.Recver

func (r AsyncRecver) Client() capnp.Client {
	return capnp.Client(r)
}

func (r AsyncRecver) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	f, release := api.Recver(r).Recv(ctx, func(ps api.Recver_recv_Params) error {
		ps.SetAsync(true)
		return nil
	})
	return Future(f), release
}

func (r AsyncRecver) AddRef() Recver {
	return AsyncRecver(r.Client().AddRef())
}

func (r AsyncRecver) Release() {
	r.Client().Release()
}
