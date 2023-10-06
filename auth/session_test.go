package auth_test

import (
	"testing"

	"capnproto.org/go/capnp/v3"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
)

func TestReleaseZeroValueSession(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, auth.Session{}.Logout,
		"should be nop")
}

func TestSessionCopy(t *testing.T) {
	t.Parallel()
	t.Helper()

	/*
		Tests that Session.AddRef() has proper copy semantics.
		This implies two things:
		  1. the dst Session struct exists in a separate arena
		     from the src.
		  2. a dst Session struct holds *new* references to the
		     capabilities in src.

		NOTE:  we do not directly test #2.  Rather, we infer that
		it is correct if our method for copying sessions is able
		to correctly copy string/int data.  In reality, we are
		testing that our (somewhat fragile) method for copying capnp
		structs is correct.
	*/

	want := newSession()
	_ = api.Session(want).Local().SetPeer("peer.ID")
	api.Session(want).Local().SetServer(9001) // routing.ID
	_ = api.Session(want).Local().SetHost("hostname")
	got := want.Clone()

	t.Run("TestDataCopied", func(t *testing.T) {
		peerID, err := api.Session(got).Local().Peer()
		require.NoError(t, err)
		assert.Equal(t, "peer.ID", peerID)

		assert.Equal(t,
			api.Session(want).Local().Server(),
			api.Session(got).Local().Server(),
			"should copy data to destination struct")

		hostname, err := api.Session(got).Local().Host()
		require.NoError(t, err)
		assert.Equal(t, "hostname", hostname)
	})

	t.Run("TestArenaSeparation", func(t *testing.T) {
		api.Session(got).Local().SetServer(42)
		assert.NotEqual(t,
			api.Session(want).Local().Server(),
			api.Session(got).Local().Server(),
			"writes to src should not mutate dst")
	})

	t.Run("TestMessageSeparation", func(t *testing.T) {
		want.Logout()
		assert.NotPanics(t, func() {
			_ = api.Session(got).Local().Server()
		}, "should exist in separate messages")
	})

}

func newSession() auth.Session {
	_, seg := capnp.NewSingleSegmentMessage(nil)
	sess, _ := api.NewRootSession(seg)
	return auth.Session(sess)
}
