package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"

	"github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/cap/anchor"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
)

type Session struct {
	Release capnp.ReleaseFunc

	View   view.View
	Root   anchor.Anchor
	PubSub pubsub.Router
}

func (sess Session) Host() host.Host {

	panic("NOT IMPLEMENTED") // host.Server{}
	return host.Host{}
}

func (sess Session) Authenticate(ctx context.Context, account Signer) Session {
	return sess
}

type Policy func(context.Context, Signer) (Session, error)

// func (auth Policy) Client() capnp.Client {
// 	term := api.Terminal_ServerToClient(auth)
// 	return capnp.Client(term)
// }

func (auth Policy) Authenticate(ctx context.Context, account Signer) (Session, error) {
	return auth(ctx, account)
}

// func (auth Policy) Login(ctx context.Context, call api.Terminal_login) error {
// 	account := call.Args().Account()

// 	sess, err := auth(ctx, func(n *Nonce) (*record.Envelope, error) {

// 	})
// 	if err != nil {
// 		return err
// 	}

// 	res, err := call.AllocResults()
// 	if err != nil {
// 		return err
// 	}

// 	res.SetPubSub()
// }

func AllowAll(ctx context.Context, h host.Host) Session {
	h = h.AddRef()

	return Session{
		Release: h.Release,
		// View: ,
	} // just(t)
}

func DenyAll() Session {
	return Session{} // nothing
}
