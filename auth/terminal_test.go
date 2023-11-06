package auth_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/api/core"
	"github.com/wetware/pkg/auth"
)

func TestTerminalServer(t *testing.T) {
	t.Parallel()
	t.Helper()

	sess := newSession()
	defer sess.Logout()

	t.Run("BaseCase:Allow", func(t *testing.T) {
		term := core.Terminal_ServerToClient(sess)
		f, release := term.Login(context.Background(), nil)
		defer release()

		s, err := f.Session().Struct()
		require.NoError(t, err, "call should succeed")
		require.NotZero(t, s, "session should be valid")
	})

	t.Run("WithPolicy:Deny", func(t *testing.T) {
		ts := sess.WithPolicy(auth.Deny("test"))

		term := core.Terminal_ServerToClient(ts)
		f, release := term.Login(context.Background(), nil)
		defer release()

		s, err := f.Session().Struct()
		require.Error(t, err, "call should fail")
		require.Zero(t, s, "session should be null")
	})
}
