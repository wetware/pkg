package auth

import (
	"context"

	"capnproto.org/go/capnp/v3"
	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/cap/view"
)

type Session api.Session

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (sess Session) AddRef() Session {
	// We start by allocating a single-segment arena.
	// This will never fail to allocate, so any errors
	// are due to undefined behavior. We use must(err)
	// to panic if an error is non-nil.
	raw, err := mkRawSession()
	must(err)

	peerID, err := api.Session(sess).Local().Peer()
	must(err)
	must(raw.Local().SetPeer(peerID))

	raw.Local().SetServer(api.Session(sess).Local().Server())

	hostname, err := api.Session(sess).Local().Host()
	must(err)
	must(raw.Local().SetHost(hostname))

	// Copy bootstrap capability.  Note how we increment the refcount.
	boot := api.Session(sess).View().AddRef()
	must(raw.SetView(boot))

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
