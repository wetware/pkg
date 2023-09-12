package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/core"
)

type SessionSetter interface {
	SetSession(api.Session) error
}

type Policy func(context.Context, SessionSetter, Session, peer.ID) error

func AllowAll(ctx context.Context, res SessionSetter, root Session, account peer.ID) error {
	sess := api.Session(root.AddRef())
	return res.SetSession(sess)
}

func OnlyAllow(allow ...peer.ID) Policy {
	allowed := make(map[peer.ID]any, len(allow))
	for _, id := range allow {
		allowed[id] = struct{}{}
	}

	return func(ctx context.Context, res SessionSetter, root Session, account peer.ID) error {
		// account not in the set of allowed accounts?
		if allowed[account] == nil {
			return errors.New("denied")
		}

		return AllowAll(ctx, res, root, account) // success
	}
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, SessionSetter, Session, peer.ID) error {
		return fmt.Errorf(reason, args...)
	}
}
