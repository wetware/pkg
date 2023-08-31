package server

import (
	"context"
	"crypto/rand"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"github.com/pkg/errors"
	api "github.com/wetware/pkg/api/auth"
	"github.com/wetware/pkg/auth"
	"golang.org/x/exp/slog"
)

func (vat Vat) NewTerminal() (auth.Terminal, capnp.ReleaseFunc) {
	term := api.Terminal_ServerToClient(vat)
	return auth.Terminal(term), term.Release
}

// Login satisfies the api/auth.Terminal_Server interface.
// This allows callers to obtain a session from a vat.
func (vat Vat) Login(ctx context.Context, call api.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	account, err := vat.roundTrip(ctx, call.Args().Account())
	if err != nil {
		return err
	}

	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	stat, err := res.NewStatus()
	if err != nil {
		return err
	}

	sess, err := vat.Auth.Authenticate(ctx, account)
	if err != nil {
		return stat.SetFailure(err.Error())
	}

	return stat.SetSuccess(sess)
}

func (vat *Vat) roundTrip(ctx context.Context, account api.Signer) (peer.ID, error) {
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
