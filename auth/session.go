package auth

import (
	"context"

	capstore_api "github.com/wetware/pkg/api/capstore"
	api "github.com/wetware/pkg/api/cluster"
	proc_api "github.com/wetware/pkg/api/process"
	"github.com/wetware/pkg/cap/capstore"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/view"
)

type Session struct {
	View     view.View
	Exec     csp.Executor
	CapStore capstore.CapStore
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
	if err := out.SetView(view.AddRef()); err != nil {
		return err
	}

	exec := proc_api.Executor(sess.Exec)
	if err := out.SetExec(exec.AddRef()); err != nil {
		return err
	}

	cs := capstore_api.CapStore(sess.CapStore)
	if err := out.SetCapStore(cs.AddRef()); err != nil {
		return err
	}

	return nil
}
