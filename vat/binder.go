package vat

import (
	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/event"
	capstore_api "github.com/wetware/pkg/api/capstore"
	cluster_api "github.com/wetware/pkg/api/cluster"
	core_api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/anchor"
	"github.com/wetware/pkg/cap/capstore"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/pubsub"
	service "github.com/wetware/pkg/cap/registry"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"go.uber.org/multierr"
)

type ViewProvider interface {
	ID() routing.ID
	View() view.View
}

type PubSubProvider interface {
	PubSub() pubsub.Router
}

type AnchorProvider interface {
	Anchor() anchor.Anchor
}

type RegistryProvider interface {
	Registry() service.Registry
}

type ExecutorProvider interface {
	Executor() csp.Executor
}

type CapStoreProvider interface {
	CapStore() capstore.CapStore
}

type binder struct {
	Emitter event.Emitter

	ViewProvider     ViewProvider
	ExecutorProvider ExecutorProvider
	CapStoreProvider CapStoreProvider
	Extra            map[string]capnp.Client
}

func (b binder) Bind(root auth.Session) error {
	if err := b.bind(root); err != nil {
		return err
	}

	return b.Emitter.Emit(root)
}

func (b binder) bind(sess auth.Session) error {
	raw := core_api.Session(sess)

	return multierr.Combine(
		b.bindView(raw),
		b.bindExec(raw),
		b.bindCapStore(raw),
		b.bindExtra(raw))
}

func (b binder) bindView(sess core_api.Session) error {
	view := b.ViewProvider.View()
	return sess.SetView(cluster_api.View(view))
}

func (b binder) bindExec(sess core_api.Session) error {
	exec := b.ExecutorProvider.Executor()
	return sess.SetExec(core_api.Executor(exec))
}

func (b binder) bindCapStore(sess core_api.Session) error {
	store := b.CapStoreProvider.CapStore()
	return sess.SetCapStore(capstore_api.CapStore(store))
}

func (b binder) bindExtra(sess core_api.Session) error {
	size := len(b.Extra)
	extra, err := sess.NewExtra(int32(size))
	if err != nil {
		return err
	}

	i := 0
	for name, c := range b.Extra {
		if err = extra.At(i).SetName(name); err != nil {
			break
		}

		if err = extra.At(i).SetClient(c); err != nil {
			break
		}

		i++
	}

	return err
}
