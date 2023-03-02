//go:generate mockgen -source=chan.go -destination=../../internal/mock/pkg/csp/chan.go -package=mock_csp

package csp

import (
	"context"

	"capnproto.org/go/capnp/v3"
	casm "github.com/wetware/casm/pkg"
	api "github.com/wetware/ww/internal/api/channel"
)

type SyncChan api.Chan

func (c SyncChan) Client() capnp.Client {
	return capnp.Client(c)
}

func (c SyncChan) Close(ctx context.Context) error {
	return Closer(c).Close(ctx)
}

func (c SyncChan) Send(ctx context.Context, v Value) error {
	return SyncSender(c).Send(ctx, v)
}

func (c SyncChan) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	return SyncRecver(c).Recv(ctx)
}

func (c SyncChan) AddRef() Chan {
	return SyncChan(c.Client().AddRef())
}

func (c SyncChan) Release() {
	c.Client().Release()
}

func (c SyncChan) AsCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	return SyncSendCloser(c).AsCloser(ctx)
}

func (c SyncChan) AsSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	return SyncSendCloser(c).AsSender(ctx)
}

func (c SyncChan) AsRecver(ctx context.Context) (Recver, capnp.ReleaseFunc) {
	f, release := api.Chan(c).AsRecver(ctx, nil)
	return SyncRecver(f.Recver()), release
}

func (c SyncChan) AsSendCloser(ctx context.Context) (SendCloser, capnp.ReleaseFunc) {
	f, release := api.Chan(c).AsSendCloser(ctx, nil)
	return SyncSendCloser(f.SendCloser()), release
}

type SyncSendCloser SyncChan

func (sc SyncSendCloser) Client() capnp.Client {
	return capnp.Client(sc)
}

func (sc SyncSendCloser) Close(ctx context.Context) error {
	return Closer(sc).Close(ctx)
}

func (sc SyncSendCloser) Send(ctx context.Context, v Value) error {
	return SyncSender(sc).Send(ctx, v)
}

func (sc SyncSendCloser) AddRef() SendCloser {
	return SyncSendCloser(sc.Client().AddRef())
}

func (sc SyncSendCloser) Release() {
	sc.Client().Release()
}

func (sc SyncSendCloser) AsSender(ctx context.Context) (Sender, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).AsSender(ctx, nil)
	return SyncSender(f.Sender()), release
}

func (sc SyncSendCloser) AsCloser(ctx context.Context) (Closer, capnp.ReleaseFunc) {
	f, release := api.SendCloser(sc).AsCloser(ctx, nil)
	return Closer(f.Closer()), release
}

type SyncSender SyncChan

func (s SyncSender) Client() capnp.Client {
	return capnp.Client(s)
}

func (s SyncSender) Send(ctx context.Context, v Value) error {
	f, release := api.Sender(s).Send(ctx, v)
	defer release()

	return casm.Future(f).Await(ctx)
}

func (s SyncSender) AddRef() Sender {
	return SyncSender(s.Client().AddRef())
}

func (s SyncSender) Release() {
	s.Client().Release()
}

type SyncRecver SyncChan

func (r SyncRecver) Client() capnp.Client {
	return capnp.Client(r)
}

func (r SyncRecver) Recv(ctx context.Context) (Future, capnp.ReleaseFunc) {
	f, release := api.Recver(r).Recv(ctx, nil)
	return Future(f), release
}

func (r SyncRecver) AddRef() Recver {
	return SyncRecver(r.Client().AddRef())
}

func (r SyncRecver) Release() {
	r.Client().Release()
}
