package system_test

import (
	"context"
	"net"
	"testing"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"golang.org/x/exp/slog"

	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/system"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystem(t *testing.T) {
	t.Parallel()

	want := auth.Session(mkRawSession())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, guest := net.Pipe()

	hostConn := rpc.NewConn(rpc.NewStreamTransport(host), &rpc.Options{
		BootstrapClient: capnp.NewClient(core.Terminal_NewServer(want)),
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default().WithGroup("server"),
		},
	})
	defer hostConn.Close()

	guestConn := rpc.NewConn(rpc.NewStreamTransport(guest), &rpc.Options{
		ErrorReporter: system.ErrorReporter{
			Logger: slog.Default().WithGroup("client"),
		},
	})
	defer guestConn.Close()

	client := guestConn.Bootstrap(ctx)
	require.NoError(t, client.Resolve(ctx), "bootstrap client should resolve")

	term := core.Terminal(client)
	defer term.Release()

	f, release := term.Login(ctx, func(call core.Terminal_login_Params) error {
		anonymous := core.Signer{}
		return call.SetAccount(anonymous)
	})
	defer release()

	res, err := f.Struct()
	require.NoError(t, err, "login should succeed")

	raw, err := res.Session()
	require.NoError(t, err, "session should be well-formed")
	require.NotZero(t, raw, "session should be populated")
	sess := auth.Session(raw)
	defer sess.Logout()

	local := raw.Local()
	assert.Equal(t, uint64(42), local.Server(),
		"should assign the correct routing.ID to the session")

	peerID, err := local.Peer()
	require.NoError(t, err, "peer.ID should be well-formed")
	assert.Equal(t, "test", peerID,
		"should assign the correct peer.ID to the session")

	hostname, err := local.Host()
	require.NoError(t, err, "hostname should be well-formed")
	assert.Equal(t, "test", hostname,
		"should assign the correct hostname to the session")

	it, release := view.View(sess.View()).Iter(ctx, view.NewQuery(view.All()))
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		// ...
	}
	assert.NoError(t, it.Err(), "iterator should not fail")
}

func mkRawSession() core.Session {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	sess, _ := core.NewRootSession(seg)
	sess.Local().SetServer(42)
	sess.Local().SetHost("test")
	sess.Local().SetPeer("test")
	return sess
}
