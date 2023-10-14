package system_test

import (
	"bytes"
	"context"
	_ "embed"
	"io"
	"strings"
	"testing"

	"github.com/stealthrocket/wazergo"
	"github.com/tetratelabs/wazero"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/wetware/pkg/system"
	"github.com/wetware/pkg/util/pipe"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/main.wasm
var src []byte

func TestSocket(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true))
	defer r.Close(ctx)
	wasi.MustInstantiate(ctx, r)

	// Instantiate wetware system socket.
	host, guest := pipe.New()
	binder := func(ctx context.Context) io.ReadWriteCloser {
		return guest
	}
	defer host.Close()

	sock, err := system.Instantiate(ctx, r, binder)
	require.NoError(t, err)
	ctx = wazergo.WithModuleInstance(ctx, sock) // bind sock to context
	defer sock.Close(ctx)

	cm, err := r.CompileModule(ctx, src)
	require.NoError(t, err)
	defer cm.Close(ctx)

	mod, err := r.InstantiateModule(ctx, cm, wazero.NewModuleConfig().
		WithStartFunctions())
	require.NoError(t, err)
	defer mod.Close(ctx)

	var g errgroup.Group
	g.Go(func() error {
		_, err := io.Copy(host, strings.NewReader("hello from host!"))
		return err
	})
	buf := new(bytes.Buffer)
	g.Go(func() error {
		_, err := io.Copy(buf, host)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, "hello from guest!", buf.String())
}

// func TestSystem(t *testing.T) {
// 	t.Parallel()

// 	want := auth.Session(mkRawSession())

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	host, guest := net.Pipe()

// 	hostConn := rpc.NewConn(rpc.NewStreamTransport(host), &rpc.Options{
// 		BootstrapClient: capnp.NewClient(core.Terminal_NewServer(want)),
// 		ErrorReporter: system.ErrorReporter{
// 			Logger: slog.Default().WithGroup("server"),
// 		},
// 	})
// 	defer hostConn.Close()

// 	guestConn := rpc.NewConn(rpc.NewStreamTransport(guest), &rpc.Options{
// 		ErrorReporter: system.ErrorReporter{
// 			Logger: slog.Default().WithGroup("client"),
// 		},
// 	})
// 	defer guestConn.Close()

// 	client := guestConn.Bootstrap(ctx)
// 	require.NoError(t, client.Resolve(ctx), "bootstrap client should resolve")

// 	term := core.Terminal(client)
// 	defer term.Release()

// 	f, release := term.Login(ctx, func(call core.Terminal_login_Params) error {
// 		anonymous := core.Signer{}
// 		return call.SetAccount(anonymous)
// 	})
// 	defer release()

// 	res, err := f.Struct()
// 	require.NoError(t, err, "login should succeed")

// 	raw, err := res.Session()
// 	require.NoError(t, err, "session should be well-formed")
// 	require.NotZero(t, raw, "session should be populated")
// 	sess := auth.Session(raw)
// 	defer sess.Logout()

// 	local := raw.Local()
// 	assert.Equal(t, uint64(42), local.Server(),
// 		"should assign the correct routing.ID to the session")

// 	peerID, err := local.Peer()
// 	require.NoError(t, err, "peer.ID should be well-formed")
// 	assert.Equal(t, "test", peerID,
// 		"should assign the correct peer.ID to the session")

// 	hostname, err := local.Host()
// 	require.NoError(t, err, "hostname should be well-formed")
// 	assert.Equal(t, "test", hostname,
// 		"should assign the correct hostname to the session")

// 	it, release := view.View(sess.View()).Iter(ctx, view.NewQuery(view.All()))
// 	defer release()

// 	for r := it.Next(); r != nil; r = it.Next() {
// 		// ...
// 	}
// 	assert.NoError(t, it.Err(), "iterator should not fail")
// }

// func mkRawSession() core.Session {
// 	_, seg := capnp.NewSingleSegmentMessage(nil)
// 	sess, _ := core.NewRootSession(seg)
// 	sess.Local().SetServer(42)
// 	sess.Local().SetHost("test")
// 	sess.Local().SetPeer("test")
// 	return sess
// }
