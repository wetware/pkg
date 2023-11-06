package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/record"
	"github.com/wetware/pkg/api/core"
)

type TerminalServer struct {
	Sess Session
	Auth Policy
}

func Negotiate(ctx context.Context, account core.Signer) (peer.ID, error) {
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

func (ts TerminalServer) Login(ctx context.Context, call core.Terminal_login) error {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*500)
	defer cancel()

	res, err := call.AllocResults()
	if err != nil {
		return fmt.Errorf("alloc results: %w", err)
	}

	account, err := Negotiate(ctx, call.Args().Account())
	if err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	return ts.Auth(ctx, res, ts.Sess, account)
}

func (ts TerminalServer) Client() core.Terminal {
	return core.Terminal_ServerToClient(ts)
}

func nonce(n Nonce) func(svr core.Signer_sign_Params) error {
	return func(call core.Signer_sign_Params) error {
		return call.SetChallenge(n[:])
	}
}
