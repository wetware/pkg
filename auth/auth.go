package auth

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/cluster"
)

type SessionCreator interface {
	NewSession() (api.Session, error)
}

type SessionBinder interface {
	BindView(api.Session) error
}

type Policy func(context.Context, SessionBinder, peer.ID, SessionCreator) error

func AllowAll(_ context.Context, root SessionBinder, _ peer.ID, call SessionCreator) error {
	sess, err := call.NewSession()
	if err != nil {
		return err
	}

	// TODO:  bind other things too...
	return root.BindView(sess)
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, SessionBinder, peer.ID, SessionCreator) error {
		return fmt.Errorf(reason, args...)
	}
}
