package auth

import (
	"context"

	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/view"
)

type Session struct {
	View view.View
}

func (sess Session) Login(ctx context.Context, call api.Terminal_login) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	out, err := res.NewSession()
	if err != nil {
		return err
	}

	// TODO:  bind other capabilities...
	view := api.View(sess.View)
	return out.SetView(view.AddRef())
}
