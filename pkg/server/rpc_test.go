package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/host"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mx "github.com/wetware/matrix/pkg"
	mock_anchor "github.com/wetware/ww/internal/test/mock/pkg/cap/anchor"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/cap/anchor"
)

const ns = "ww.test"

func TestRPC_conn_lifecycle(t *testing.T) {
	t.Parallel()
	t.Helper()

	/*
	 * The purpose of this test is to ensure that RPC connections
	 * and their associated libp2p streams are released in a timely
	 * manner when clients disconnect.  It was added in response to
	 * an observed bug in which connections persisted despite calls
	 * to client.Node.Close.
	 */

	t.Run("ClientHangUp", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			sim = mx.New(ctx)
			h0  = sim.MustHost(ctx)
			h1  = sim.MustHost(ctx)
		)

		defer func() {
			require.NoError(t, h0.Close(), "host h0 should close cleanly")
			require.NoError(t, h1.Close(), "host h1 should close cleanly")
		}()

		log := logtest.NewMockLogger(ctrl)
		log.EXPECT().
			With(gomock.Any()).
			Return(log).
			AnyTimes()
		log.EXPECT().
			WithField(gomock.Any(), gomock.Any()).
			Return(log).
			AnyTimes()

		// <-conn.Done()
		log.EXPECT().
			Debug(renderEq("client hung up")).
			Times(1)
		c := mock_anchor.NewMockCluster(ctrl)
		c.EXPECT().
			String().
			Return(ns).
			AnyTimes()
		c.EXPECT().
			Close().
			Times(1)

		cs := newCapSet(
			anchor.New(c),
			nil)
		cs.registerRPC(h0, log)
		defer cs.Close()
		defer cs.unregisterRPC(h0)

		err := h1.Connect(ctx, *host.InfoFromHost(h0))
		require.NoError(t, err,
			"test invariant violated:  connection must succeed")

		s, err := h1.NewStream(ctx, h0.ID(), ww.Subprotocol(ns))
		require.NoError(t, err, "should successfully open stram")

		conn := rpc.NewConn(rpc.NewStreamTransport(s), nil)

		client := conn.Bootstrap(ctx)
		err = client.Resolve(ctx)
		require.NoError(t, err, "should resolve successfully")

		time.Sleep(time.Millisecond * 10)

		err = conn.Close()
		require.NoError(t, err, "conn should close cleanly")

		time.Sleep(time.Millisecond * 10)
	})

	t.Run("Shutdown", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var (
			sim = mx.New(ctx)
			h0  = sim.MustHost(ctx)
			h1  = sim.MustHost(ctx)
		)

		defer func() {
			require.NoError(t, h0.Close(), "host h0 should close cleanly")
			require.NoError(t, h1.Close(), "host h1 should close cleanly")
		}()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		log := logtest.NewMockLogger(ctrl)
		log.EXPECT().
			With(gomock.Any()).
			Return(log).
			AnyTimes()
		log.EXPECT().
			WithField(gomock.Any(), gomock.Any()).
			Return(log).
			AnyTimes()

		// <-cq
		log.EXPECT().
			Debug(renderEq("shutting down")).
			Times(1)

		c := mock_anchor.NewMockCluster(ctrl)
		c.EXPECT().
			String().
			Return(ns).
			AnyTimes()
		c.EXPECT().
			Close().
			Times(1)

		cs := newCapSet(
			anchor.New(c),
			nil)
		cs.registerRPC(h0, log)
		defer cs.unregisterRPC(h0)

		err := h1.Connect(ctx, *host.InfoFromHost(h0))
		require.NoError(t, err,
			"test invariant violated:  connection must succeed")

		s, err := h1.NewStream(ctx, h0.ID(), ww.Subprotocol(ns))
		require.NoError(t, err, "should successfully open stram")

		conn := rpc.NewConn(rpc.NewStreamTransport(s), nil)
		defer conn.Close()

		client := conn.Bootstrap(ctx)
		err = client.Resolve(ctx)
		require.NoError(t, err, "should resolve successfully")

		time.Sleep(time.Millisecond * 10)

		err = cs.Close()
		require.NoError(t, err, "capSet should close cleanly")
	})
}

func TestCapSet(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Close", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		c := mock_anchor.NewMockCluster(ctrl)
		c.EXPECT().
			String().
			Return(ns).
			AnyTimes()
		c.EXPECT().
			Close().
			Return(nil).
			Times(1)

		cs := newCapSet(
			anchor.New(c),
			nil)

		err := cs.Close()
		assert.NoError(t, err, "should close cleanly")

		err = cs.Close()
		assert.EqualError(t, err, "already closed")
	})
}

type renderEq string

func (m renderEq) String() string {
	return fmt.Sprintf("\"%s\"", string(m))
}

func (m renderEq) Matches(x interface{}) bool {
	return fmt.Sprintf("%s", x) == string(m)
}
