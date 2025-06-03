package vat

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"
	"go.uber.org/multierr"
	"golang.org/x/exp/slog"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
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

// Server provides the Host capability.
type Server struct {
	NS               string
	Host             local.Host
	Auth             auth.Policy
	ViewProvider     ViewProvider
	ExecutorProvider ExecutorProvider
	CapStoreProvider CapStoreProvider
	Extra            map[string]capnp.Client

	once sync.Once
	ch   chan network.Stream
}

func (svr *Server) setup() {
	svr.once.Do(func() {
		svr.ch = make(chan network.Stream)
	})
}

// Close the vat.Network implementation.  Note that Close()
// does not affect ongoing requests, nor does it release any
// capabilities.
func (svr *Server) Close() error {
	svr.setup()
	close(svr.ch)

	for _, c := range svr.Extra {
		c.Release()
	}

	return nil
}

func (svr *Server) Export() capnp.Client {
	return capnp.NewClient(core_api.Terminal_NewServer(svr))
}

func (svr *Server) NewRootSession() (core_api.Session, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	sess, err := core_api.NewRootSession(seg) // TODO(optimization):  non-root?
	if err != nil {
		return core_api.Session{}, err
	}

	routingID := svr.ViewProvider.ID()
	sess.Local().SetServer(uint64(routingID))

	hostname, err := sess.Local().Host()
	if err != nil {
		return core_api.Session{}, err
	}

	// Write session data
	err = multierr.Combine(
		// Local data
		// TODO(soon):  sess.Local().SetNamespace(svr.NS),
		sess.Local().SetHost(hostname),
		sess.Local().SetPeer(string(svr.Host.ID())),

		// Capabilities
		svr.BindView(sess),
		svr.BindExec(sess),
		svr.BindCapStore(sess),
		svr.BindExtra(sess),
	)

	return sess, err
}

func (svr *Server) Login(ctx context.Context, call core_api.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := call.AllocResults()
	if err != nil {
		return fmt.Errorf("alloc results: %w", err)
	}

	account, err := svr.Negotiate(ctx, call.Args().Account())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	root, err := svr.NewRootSession()
	if err != nil {
		return err
	}

	return svr.Auth(ctx, res, auth.Session(root), account)
}

func (svr *Server) Negotiate(ctx context.Context, account cluster_api.Signer) (peer.ID, error) {
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
	// we should assume it'svr an attack.
	if u != n {
		slog.Warn("login failed",
			"error", "nonce mismatch",
			"want", n,
			"got", u)

		return "", errors.New("nonce mismatch")
	}

	return id, nil
}

func nonce(n auth.Nonce) func(svr cluster_api.Signer_sign_Params) error {
	return func(call cluster_api.Signer_sign_Params) error {
		return call.SetChallenge(n[:])
	}
}

func (svr *Server) BindView(sess core_api.Session) error {
	view := svr.ViewProvider.View()
	return sess.SetView(cluster_api.View(view))
}

func (svr *Server) BindExec(sess core_api.Session) error {
	exec := svr.ExecutorProvider.Executor()
	return sess.SetExec(core_api.Executor(exec))
}

func (svr *Server) BindCapStore(sess core_api.Session) error {
	store := svr.CapStoreProvider.CapStore()
	return sess.SetCapStore(capstore_api.CapStore(store))
}

func (svr *Server) BindExtra(sess core_api.Session) error {
	size := len(svr.Extra)
	extra, err := sess.NewExtra(int32(size))
	if err != nil {
		return err
	}

	i := 0
	for name, c := range svr.Extra {
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
