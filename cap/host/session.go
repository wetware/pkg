package host

import (
	"capnproto.org/go/capnp/v3"
	anchor_api "github.com/wetware/pkg/api/anchor"
	api "github.com/wetware/pkg/api/cluster"
	pubsub_api "github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/cap/pubsub"
	"github.com/wetware/pkg/cap/view"
	"go.uber.org/multierr"
)

type Client[T ~capnp.ClientKind] interface {
	AddRef() T
	Release()
}

type Session struct {
	View   view.View
	Router pubsub.Router
}

func (sess *Session) Bind(res api.Host_login_Results) error {
	return multierr.Combine(
		bind[api.View](res.SetView, api.View(sess.View)),
		bind[pubsub_api.Router](res.SetPubSub, pubsub_api.Router(sess.Router)),
		bind[anchor_api.Anchor](res.SetRoot, anchor_api.Anchor{}),
		/* TODO: ... */)

}

func bind[T ~capnp.ClientKind](set func(T) error, t Client[T]) error {
	return set(t.AddRef())
}
