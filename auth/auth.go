package auth

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/cluster"
)

type SessionSetter interface {
	SetSession(api.Session) error
}

type Policy func(context.Context, SessionSetter, Session, peer.ID) error

func AllowAll(ctx context.Context, res SessionSetter, root Session, account peer.ID) error {
	sess := api.Session(root.AddRef())
	return res.SetSession(sess)
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, SessionSetter, Session, peer.ID) error {
		return fmt.Errorf(reason, args...)
	}
}
