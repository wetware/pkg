//go:generate mockgen -source=host.go -destination=test/host.go -package=test_host

package host

import (
	"context"

	"capnproto.org/go/capnp/v3"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/view"
)

/*----------------------------*
|                             |
|    Client Implementations   |
|                             |
*-----------------------------*/

type Host api.Host

func (h Host) AddRef() Host {
	return Host(capnp.Client(h).AddRef())
}

func (h Host) Release() {
	capnp.Client(h).Release()
}

func (h Host) Login(ctx context.Context, account api.Signer) (Session, error) {
	f, release := api.Host(h).Login(ctx, func(call api.Host_login_Params) error {
		return call.SetAccount(account)
	})
	defer release()

	res, err := f.Struct()
	return Session{
		View: view.View(res.View()),
		// TODO:  add the rest...
	}, err
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

type AuthPolicy func(ctx context.Context, call api.Host_login) error

func (f AuthPolicy) Client() capnp.Client {
	client := api.Host_ServerToClient(f)
	return capnp.Client(client)
}

func (f AuthPolicy) Login(ctx context.Context, call api.Host_login) error {
	return f(ctx, call)
}
