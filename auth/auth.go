package auth

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"

	api "github.com/wetware/pkg/api/cluster"
)

type SessionCreator interface {
	NewSession() (api.Session, error)
	SetSession(api.Session) error
}

type SessionBinder interface {
	BindView(api.Session) error
	BindExec(api.Session) error
	BindCapStore(api.Session) error
}

type Policy func(context.Context, SessionBinder, peer.ID, SessionCreator) error

func AllowAll(_ context.Context, root SessionBinder, _ peer.ID, call SessionCreator) error {
	sess, err := call.NewSession()
	if err != nil {
		return err
	}

	if err = root.BindView(sess); err != nil {
		return fmt.Errorf("view: %w", err)
	}

	if err = root.BindExec(sess); err != nil {
		return fmt.Errorf("exec: %w", err)
	}

	if err = root.BindCapStore(sess); err != nil {
		return fmt.Errorf("capstore: %w", err)
	}

	return call.SetSession(sess)
}

func Deny(reason string, args ...any) Policy {
	return func(context.Context, SessionBinder, peer.ID, SessionCreator) error {
		return fmt.Errorf(reason, args...)
	}
}
