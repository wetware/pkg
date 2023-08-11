package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/cluster"
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

func (s Session) View() view.View {
	return view.View(s.AuthProvider_provide_Results_Future.View())
}
