package auth

import (
	"context"

	"github.com/wetware/pkg/api/anchor"
	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/cap/host"
	"go.uber.org/multierr"
	"golang.org/x/exp/slog"
)

// AllowAll is a policy that grants unrestricted access to h.
// Callers SHOULD NOT use AllowAll if they can avoid it.
func AllowAll(h host.Host) api.AuthProvider {
	return Policy(just{h})
}

// DenyAll is a policy that does not grant access to h.  It is
// RECOMMENDED to use DenyAll by default.
func DenyAll() api.AuthProvider {
	return Policy(nothing{}) // null client
}

// // SharedSecret requires that both parties share knowledge of
// // a secret.  Secrets should be produced by a strong CSPRNG.
// // The secret is not transmitted.
// func SharedSecret(h api.Host_Server, secret []byte) capnp.Client {
// 	return capnp.Client(api.AuthProvider_ServerToClient(server{secret}))
// }

func Policy(s api.AuthProvider_Server) api.AuthProvider {
	return api.AuthProvider_ServerToClient(s)
}

// just{h} === Just(h)
type just struct {
	host.Host
}

func (j just) Provide(ctx context.Context, call api.AuthProvider_provide) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	slog.Default().Info("view")
	view, release := j.View(ctx)
	defer release()

	slog.Default().Info("root")
	root, release := j.Root(ctx)
	defer release()

	slog.Default().Info("pubsub")
	router, release := j.PubSub(ctx)
	defer release()

	return multierr.Combine(
		res.SetView(api.View(view.AddRef())),
		res.SetRoot(anchor.Anchor(root.AddRef())),
		res.SetPubSub(pubsub.Router(router.AddRef())))

}

// nothing{} === Nothing === Maybe(nil)
type nothing struct{}

func (n nothing) Provide(context.Context, api.AuthProvider_provide) error {
	return nil
}
