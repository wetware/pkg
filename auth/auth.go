package auth

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	api "github.com/wetware/pkg/api/auth"
)

type Policy func(context.Context, peer.ID) (api.Session, error)

func Deny(reason string, args ...any) Policy {
	return func(context.Context, peer.ID) (api.Session, error) {
		return api.Session{}, fmt.Errorf(reason, args...)
	}
}

func (auth Policy) Authenticate(ctx context.Context, account peer.ID) (api.Session, error) {
	if auth == nil {
		// Default to denying all auth requests.
		return Deny("no policy").Authenticate(ctx, account)
	}

	return auth(ctx, account)
}
