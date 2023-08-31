package auth

import (
	"context"
	"fmt"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/auth"
	"github.com/wetware/pkg/cap/view"
)

type Session interface {
	Err() error
	Close() error
	View() view.View
}

type Terminal api.Terminal

func (t Terminal) Login(ctx context.Context, account Signer) (Session, error) {
	if account == nil {
		return authFailure("no account specified"), nil
	}

	f, release := api.Terminal(t).Login(ctx, account.Bind(ctx))

	res, err := f.Struct()
	if err != nil {
		defer release()
		return nil, err
	}

	stat, err := res.Status()
	if err != nil {
		defer release()
		return nil, err
	}

	switch stat.Which() {
	case api.Terminal_Status_Which_success:
		sess, err := stat.Success()
		if err != nil {
			defer release()
			return nil, err
		}
		return session{Session: sess, release: release}, nil

	case api.Terminal_Status_Which_failure:
		defer release()
		reason, err := stat.Failure()
		return authFailure(reason), err

	default:
		defer release()
		return nil, fmt.Errorf("unrecognized status: %d", stat.Which())
	}

}

type session struct {
	api.Session
	release capnp.ReleaseFunc
}

func (sess session) Err() error {
	return nil
}

func (sess session) Close() error {
	sess.release()
	return nil
}

func (sess session) View() view.View {
	return view.View(sess.Session.View())
}

type authFailure string

func (f authFailure) Err() error {
	return f
}

func (f authFailure) Error() string {
	return string(f)
}

func (authFailure) Close() error {
	return nil
}

func (f authFailure) Client() capnp.Client {
	return capnp.ErrorClient(f)
}

func (f authFailure) View() view.View {
	return view.View(f.Client())
}

// func (f authFailure)
