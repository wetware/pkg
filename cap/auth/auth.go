package auth

import (
	"context"
	"errors"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/pkg/api/anchor"
	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
)

type Provider api.AuthProvider

func (p Provider) AddRef() Provider {
	return Provider(api.AuthProvider(p).AddRef())
}

func (p Provider) Release() {
	api.AuthProvider(p).Release()
}

func (p Provider) Provide(ctx context.Context, account Signer) (Session, capnp.ReleaseFunc) {
	f, release := api.AuthProvider(p).Provide(ctx, func(call api.AuthProvider_provide_Params) error {
		return call.SetAccount(api.Signer(account))
	})

	return Session{f}, release
}

type Session struct {
	api.AuthProvider_provide_Results_Future
}

func (sess Session) Host() host.Host {
	return sessionServer{
		view: view.View(sess.View().AddRef()),

		// TODO(soon):  add remaining capabilities
		// pubsub: ...
		// root: ...
	}.Host()
}

type sessionServer struct {
	view   view.View
	pubsub pubsub.Router
	root   anchor.Anchor
	// registry ...
	// executor ...
}

func (p sessionServer) Shutdown() {
	p.view.Release()
}

func (p sessionServer) Host() host.Host {
	return host.Host(api.Host_ServerToClient(p))
}

func (p sessionServer) View(ctx context.Context, call api.Host_view) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetView(api.View(p.view).AddRef())
}

func (p sessionServer) PubSub(ctx context.Context, call api.Host_pubSub) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetPubSub(pubsub.Router(p.pubsub).AddRef())
}

func (p sessionServer) Root(ctx context.Context, call api.Host_root) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetRoot(anchor.Anchor(p.root).AddRef())
}

func (p sessionServer) Registry(ctx context.Context, call api.Host_registry) error {
	return errors.New("sessionServer.Registry: NOT IMPLEMENTED") // TODO(soon)
}

func (p sessionServer) Executor(ctx context.Context, call api.Host_executor) error {
	return errors.New("sessionServer.Executor: NOT IMPLEMENTED") // TODO(soon)
}
