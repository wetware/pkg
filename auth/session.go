package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"
	"github.com/wetware/pkg/api/capstore"
	api "github.com/wetware/pkg/api/cluster"
	"github.com/wetware/pkg/cap/csp"
	"github.com/wetware/pkg/cap/view"
)

type Session api.Session

func (sess Session) AddRef() Session {
	raw, err := mkRawSession()
	if err != nil {
		panic(err) // single-segment arena should never fail to allocate
	}

	peerID, _ := api.Session(sess).Local().Peer()
	_ = raw.Local().SetPeer(peerID)

	raw.Local().SetServer(api.Session(sess).Local().Server())

	hostname, _ := api.Session(sess).Local().Host()
	_ = raw.Local().SetHost(hostname)

	// copy capabilities; we MUST increment the refcount.
	raw.SetView(api.Session(sess).View().AddRef())
	raw.SetExec(api.Session(sess).Exec().AddRef())
	raw.SetCapStore(api.Session(sess).CapStore().AddRef())

	return Session(raw)
}

// Terminate the session by releasing the message, which releases
// each entry in the cap table.
func (sess Session) Terminate() {
	api.Session(sess).Message().Release()
}

// Login allows the session to be served as a Terminal.  It provides full
// access to the Session object.  Use carefully.
func (sess Session) Login(ctx context.Context, call api.Terminal_login) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	return res.SetSession(api.Session(sess))
}

func (sess Session) View() view.View {
	client := api.Session(sess).View()
	return view.View(client)
}

func (sess Session) Exec() csp.Executor {
	client := api.Session(sess).Exec()
	return csp.Executor(client)
}

func (sess Session) CapStore() capstore.CapStore {
	client := api.Session(sess).CapStore()
	return capstore.CapStore(client)
}

// mkRawSession allocates a new api.Session.  Error is always nil.
func mkRawSession() (api.Session, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	return api.NewRootSession(seg) // TODO(performance):  non-root message
}
