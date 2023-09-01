package auth

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
	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/host"
)

type Terminal api.Terminal

func (t Terminal) AddRef() Terminal {
	return Terminal(capnp.Client(t).AddRef())
}

func (t Terminal) Release() {
	capnp.Client(t).Release()
}

func (t Terminal) Login(ctx context.Context, account Signer) (host.Host, error) {
	f, release := api.Terminal(t).Login(ctx, account.Bind(ctx))
	defer release()

	res, err := f.Struct()
	if err != nil {
		return host.Host{}, err
	}

	return host.Host(res.Host()).AddRef(), nil
}

type TerminalServer struct {
	Host   host.Host
	Policy Policy
}

func (term TerminalServer) Login(ctx context.Context, call api.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	account, err := term.Negotiate(ctx, call.Args().Account())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	host, release := term.Policy.Authenticate(ctx, term.Host, account)
	defer release()

	return res.SetHost(host.AddRef())
}

func (term TerminalServer) Negotiate(ctx context.Context, account api.Signer) (peer.ID, error) {
	var n Nonce
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

	var u Nonce
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

func nonce(n Nonce) func(s api.Signer_sign_Params) error {
	return func(call api.Signer_sign_Params) error {
		return call.SetChallenge(n[:])
	}
}
