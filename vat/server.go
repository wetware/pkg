package vat

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"
	"golang.org/x/exp/slog"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/view"
)

// Server provides the Host capability.
type Server struct {
	NS     string
	Host   local.Host
	Auth   auth.Policy
	OnJoin interface {
		Emit(any) error
	}

	Root    auth.Session
	Cluster interface {
		View() view.View
	}

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
	return nil
}

func (svr *Server) Export() capnp.Client {
	return capnp.NewClient(core.Terminal_NewServer(svr))
}

func (svr *Server) Login(ctx context.Context, call core.Terminal_login) error {
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

	// svr.Auth is responsible for incrementing the refcount on the
	// root session.
	return svr.Auth(ctx, res, svr.Root, account)
}

func (svr *Server) Negotiate(ctx context.Context, account core.Signer) (peer.ID, error) {
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

func nonce(n auth.Nonce) func(svr core.Signer_sign_Params) error {
	return func(call core.Signer_sign_Params) error {
		return call.SetChallenge(n[:])
	}
}
