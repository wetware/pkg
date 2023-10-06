package auth

import (
	"context"
	"log/slog"

	"capnproto.org/go/capnp/v3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
)

type Session core.Session

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (sess Session) Clone() Session {
	// We start by allocating a single-segment arena.
	// This will never fail to allocate, so any errors
	// are due to undefined behavior. We use must(err)
	// to panic if an error is non-nil.
	raw, err := mkRawSession()
	must(err)

	peerID, err := core.Session(sess).Local().Peer()
	must(err)
	must(raw.Local().SetPeer(peerID))

	raw.Local().SetServer(core.Session(sess).Local().Server())

	hostname, err := core.Session(sess).Local().Host()
	must(err)
	must(raw.Local().SetHost(hostname))

	// Copy bootstrap capability.  Note how we increment the refcount.
	boot := core.Session(sess).View().AddRef()
	must(raw.SetView(boot))

	return Session(raw)
}

// Logout of the session by releasing the message, which releases
// each entry in the cap table.
func (sess Session) Logout() {
	message := core.Session(sess).Message()
	if message != nil {
		message.Release()
	} else {
		slog.Debug("noop",
			"reason", "null message")
	}
}

// Login allows the session to be served as a Terminal.  It provides full
// access to the Session object.  Use carefully.
func (sess Session) Login(ctx context.Context, call core.Terminal_login) error {
	res, err := call.AllocResults()
	if err != nil {
		return err
	}

	if err := res.SetSession(core.Session(sess)); err != nil {
		return err
	}

	slog.Info("user logged in", // TODO:  who?
		"vat", sess.Vat(),
		"peer", sess.Peer(),
		"hostname", sess.Hostname(),
		"view", capnp.Client(sess.View()))

	return nil
}

func (sess Session) View() view.View {
	client := core.Session(sess).View()
	return view.View(client)
}

func (sess Session) Vat() routing.ID {
	local := core.Session(sess).Local()
	return routing.ID(local.Server())
}

func (sess Session) Peer() peer.ID {
	local := core.Session(sess).Local()

	s, err := local.Peer()
	if err != nil {
		slog.Debug("failed to access field",
			"reason", err)
	}

	return peer.ID(s)
}

func (sess Session) Hostname() string {
	local := core.Session(sess).Local()

	s, err := local.Host()
	if err != nil {
		slog.Debug("failed to access field",
			"reason", err)
	}

	return s
}

// func (sess Session) Imports() (map[string]capnp.Client, capnp.ReleaseFunc) {
// 	extra, err := core.Session(sess).Extra()
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
// 	extra, err := core.Session(sess).Extra()

// 	for i := 0; i < extra.Len(); i++ {
// 		key, err := extra.At(i).Name()
// 		if key == name || err != nil {
// 			client := extra.At(i).Client()
// 			return client.AddRef(), err
// 		}
// 	}

// 	return capnp.Client{}, err
// }

// mkRawSession allocates a new core.Session.  Error is always nil.
func mkRawSession() (core.Session, error) {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	return core.NewRootSession(seg) // TODO(performance):  non-root message
}
