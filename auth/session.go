package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/cap/capstore"
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
	extra, err := api.Session(sess).Extra()
	if err == nil && extra.Len() > 0 {
		err := api.Session(sess).SetExtra(extra)
		if err != nil {
			panic(err)
		}
	}

	return Session(raw)
}

// Release the session by releasing the message, which releases
// each entry in the cap table.
func (sess Session) Release() {
	if sess != (Session{}) {
		api.Session(sess).Message().Release()
	}
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

// func (sess Session) Imports() (map[string]capnp.Client, capnp.ReleaseFunc) {
// 	extra, err := api.Session(sess).Extra()
// 	if err != nil || extra.Len() == 0 {
// 		return nil, func() {}
// 	}

// 	imports := make(map[string]capnp.Client, extra.Len())
// 	for i := 0; i < extra.Len(); i++ {
// 		name, err := extra.At(i).Name()
// 		if err == nil {
// 			imports[name] = extra.At(i).Client().AddRef()
// 		}
// 	}

// 	return imports, func() {
// 		for _, c := range imports {
// 			c.Release()
// 		}
// 	}
// }

// func (sess Session) Import(name string) (capnp.Client, error) {
// 	extra, err := api.Session(sess).Extra()

// 	for i := 0; i < extra.Len(); i++ {
// 		key, err := extra.At(i).Name()
// 		if key == name || err != nil {
// 			client := extra.At(i).Client()
// 			return client.AddRef(), err
// 		}
// 	}

// 	return capnp.Client{}, err
// }

// mkRawSession allocates a new api.Session.  Error is always nil.
func mkRawSession() (api.Session, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	return api.NewRootSession(seg) // TODO(performance):  non-root message
}
