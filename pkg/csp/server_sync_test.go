package csp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/csp"
	"golang.org/x/sync/errgroup"
)

func TestSyncServer(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("TypeEnforced", func(t *testing.T) {
		t.Parallel()

		ch := csp.NewChan(&csp.SyncServer{})
		ach := csp.AsyncChan(ch.Client())

		err := ach.Send(context.Background(), csp.Text("fail"))
		assert.EqualError(t, err,
			"channel.capnp:Sender.send: async call to sync channel")

		f, release := ach.Recv(context.Background())
		defer release()

		_, err = f.Text()
		assert.EqualError(t, err,
			"channel.capnp:Recver.recv: async call to sync channel")
	})

	t.Run("SenderFirst", func(t *testing.T) {
		t.Parallel()

		const want = "hello, world!"
		ch := csp.NewChan(&csp.SyncServer{})

		sync := make(chan struct{})
		go func() {
			close(sync)

			err := ch.Send(context.Background(), csp.Text(want))
			require.NoError(t, err)
		}()

		// wait for the send call to be in flight
		<-sync

		f, release := ch.Recv(context.Background())
		defer release()

		got, err := f.Text()
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("RecverFirst", func(t *testing.T) {
		t.Parallel()

		const want = "hello, world!"
		ch := csp.NewChan(&csp.SyncServer{})

		sync := make(chan struct{})
		go func() {
			f, release := ch.Recv(context.Background())
			defer release()

			// wait for the recv call to be in flight
			close(sync)

			got, err := f.Text()
			require.NoError(t, err)
			require.Equal(t, want, got)
		}()

		<-sync

		err := ch.Send(context.Background(), csp.Text(want))
		require.NoError(t, err)
	})

	t.Run("ConcurrentSendRecv", func(t *testing.T) {
		t.Parallel()

		var g errgroup.Group

		ch := csp.NewChan(&csp.SyncServer{})

		want := []string{
			"alpha",
			"bravo",
			"charlie",
			"delta",
			"echo",
			"fox",
			"golf",
			"hotel",
			"india"}
		got := make([]string, len(want))

		sender := func(msg string) func() error {
			return func() error {
				return ch.Send(context.Background(), csp.Text(msg))
			}
		}

		recver := func(i int, msg string) func() error {
			return func() error {
				f, release := ch.Recv(context.Background())
				defer release()

				msg, err := f.Text()
				got[i] = msg
				return err
			}
		}

		for i, m := range want {
			g.Go(sender(m))
			g.Go(recver(i, m))
		}

		require.NoError(t, g.Wait())
		require.ElementsMatch(t, want, got)
	})
}
