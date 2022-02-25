package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	api "github.com/wetware/ww/internal/api/pubsub"
)

func TestHandler(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("Handle", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ms := make(chan []byte, 1)
		h := handler{
			ms:      ms,
			release: func() {},
		}

		c := api.Topic_Handler_ServerToClient(h, nil)
		defer c.Release()

		f, release := c.Handle(ctx, func(ps api.Topic_Handler_handle_Params) error {
			return ps.SetMsg([]byte("test"))
		})
		defer release()

		_, err := f.Struct()
		assert.NoError(t, err, "call to Handle should succeed")
		assert.Equal(t, "test", string(<-ms), "unexpected message")
	})

	t.Run("Release", func(t *testing.T) {
		t.Parallel()

		var (
			called bool
			ms     = make(chan []byte, 1)
		)

		h := handler{
			ms:      ms,
			release: func() { called = true },
		}

		c := api.Topic_Handler_ServerToClient(h, nil)
		c.Release()

		require.True(t, called, "should call release function")

		_, ok := <-ms
		assert.False(t, ok, "should close message channel when released")
	})
}
