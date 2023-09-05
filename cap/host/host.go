//go:generate mockgen -source=host.go -destination=test/host.go -package=test_host

package host

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"capnproto.org/go/capnp/v3"
	"golang.org/x/exp/slog"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"github.com/tetratelabs/wazero"
	api "github.com/wetware/pkg/api/cluster"
	pubsub_api "github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/anchor"
	"github.com/wetware/pkg/cap/capstore"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/pubsub"
	service "github.com/wetware/pkg/cap/registry"
	"github.com/wetware/pkg/cap/view"
)

/*----------------------------*
|                             |
|    Client Implementations   |
|                             |
*-----------------------------*/

type Host api.Host

func (h Host) Resolve(ctx context.Context) error {
	return capnp.Client(h).Resolve(ctx)
}

func (h Host) AddRef() Host {
	return Host(capnp.Client(h).AddRef())
}

func (h Host) Release() {
	capnp.Client(h).Release()
}

func (h Host) View(ctx context.Context) (view.View, capnp.ReleaseFunc) {
	f, release := api.Host(h).View(ctx, nil)
	return view.View(f.View()), release
}

func (h Host) PubSub(ctx context.Context) (pubsub.Router, capnp.ReleaseFunc) {
	f, release := api.Host(h).PubSub(ctx, nil)
	return pubsub.Router(f.PubSub()), release
}

func (h Host) Root(ctx context.Context) (anchor.Anchor, capnp.ReleaseFunc) {
	f, release := api.Host(h).Root(ctx, nil)
	return anchor.Anchor(f.Root()), release
}

func (h Host) Registry(ctx context.Context) (service.Registry, capnp.ReleaseFunc) {
	f, release := api.Host(h).Registry(ctx, nil)
	return service.Registry(f.Registry()), release
}

func (h Host) Executor(ctx context.Context) (csp.Executor, capnp.ReleaseFunc) {
	f, release := api.Host(h).Executor(ctx, nil)
	return csp.Executor(f.Executor()), release
}

/*---------------------------*
|                            |
|    Server Implementation   |
|                            |
*----------------------------*/

type ViewProvider interface {
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

// Server provides the Host capability.
type Server struct {
	Auth           auth.Policy
	ViewProvider   ViewProvider
	PubSubProvider PubSubProvider
	RuntimeConfig  wazero.RuntimeConfig

	pubsub *pubsub.Server
}

func (s *Server) Host() api.Host {
	return api.Host_ServerToClient(s)
}

func (s *Server) Export() capnp.Client {
	return capnp.NewClient(api.Terminal_NewServer(s))
}

func (s *Server) Login(ctx context.Context, call api.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	account, err := s.Negotiate(ctx, call.Args().Account())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	host, release := s.Auth.Authenticate(ctx, s.Host(), account)
	defer release()

	return res.SetHost(host.AddRef())
}

func (s *Server) Negotiate(ctx context.Context, account api.Signer) (peer.ID, error) {
	var n auth.Nonce
	if _, err := rand.Read(n[:]); err != nil {
		panic(err) // unreachable
	}

	f, release := account.Sign(ctx, nonce(n))
	defer release()

	res, err := f.Struct()
	if err != nil {
		return "", err
	}

	b, err := res.Signed()
	if err != nil {
		return "", err
	}

	var u auth.Nonce
	e, err := record.ConsumeTypedEnvelope(b, &u)
	if err != nil {
		return "", err
	}

	// Derive the peer.ID from the signed nonce.  This gives us the
	// identity of the account that is trying to log in.
	id, err := peer.IDFromPublicKey(e.PublicKey)
	if err != nil {
		return "", err
	}

	// Make sure the record is for the nonce we sent.  If it isn't,
	// we should assume it's an attack.
	if u != n {
		slog.Warn("login failed",
			"error", "nonce mismatch",
			"want", n,
			"got", u)

		return "", errors.New("nonce mismatch")
	}

	return id, nil
}

func nonce(n auth.Nonce) func(s api.Signer_sign_Params) error {
	return func(call api.Signer_sign_Params) error {
		return call.SetChallenge(n[:])
	}
}

func (s *Server) View(_ context.Context, call api.Host_view) error {
	res, err := call.AllocResults()
	if err == nil {
		view := s.ViewProvider.View()
		err = res.SetView(api.View(view))
	}

	return err
}

func (s *Server) PubSub(_ context.Context, call api.Host_pubSub) error {
	res, err := call.AllocResults()
	if err == nil {
		router := pubsub_api.Router_ServerToClient(s.pubsub)
		err = res.SetPubSub(router)
	}

	return err
}

func (s *Server) Root(_ context.Context, call api.Host_root) error {
	panic("NOT IMPLEMENTED")
	// res, err := call.AllocResults()
	// if err == nil {
	// 	err = res.SetRoot(anchor_api.Anchor(s.AnchorProvider.Anchor()))
	// }

	// return err
}

func (s *Server) Registry(_ context.Context, call api.Host_registry) error {
	panic("NOT IMPLEMENTED")
	// res, err := call.AllocResults()
	// if err == nil {
	// 	registry := s.RegistryProvider.Registry()
	// 	err = res.SetRegistry(reg_api.Registry(registry))
	// }

	// return err
}

func (s *Server) Executor(_ context.Context, call api.Host_executor) error {
	panic("NOT IMPLEMENTED")
	// res, err := call.AllocResults()
	// if err == nil {
	// 	e := s.ExecutorProvider.Executor()
	// 	err = res.SetExecutor(process_api.Executor(e))
	// }
	// return err
}

func (s *Server) CapStore(_ context.Context, call api.Host_capStore) error {
	panic("NOT IMPLEMENTED")
	// res, err := call.AllocResults()
	// if err == nil {
	// 	c := s.CapStoreProvider.CapStore()
	// 	err = res.SetCapStore(capstore_api.CapStore(c))
	// }
	// return err
}
