package client_test

import (
	"context"
	"errors"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mock_client "github.com/wetware/ww/internal/mock/pkg/client"
	"github.com/wetware/ww/pkg/client"
)

func TestHostConn(t *testing.T) {
	t.Parallel()
	t.Helper()

	t.Run("NullClient", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		conn := mock_client.NewMockConn(ctrl)
		conn.EXPECT().
			Bootstrap(gomock.Any()).
			Return(capnp.Client{}).
			Times(1)

		hconn := &client.HostConn{
			Conn: conn,
		}

		c := hconn.Bootstrap(context.Background())
		assert.Equal(t, capnp.Client{}, c,
			"should bootstrap when cached client is null")
	})

	t.Run("ErrorClient", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ec := capnp.ErrorClient(errors.New("test"))

		conn := mock_client.NewMockConn(ctrl)
		conn.EXPECT().
			Bootstrap(gomock.Any()).
			Return(ec).
			Times(2)

		hconn := &client.HostConn{
			Conn: conn,
		}

		// call once to cache the error client...
		_ = hconn.Bootstrap(context.Background())

		// ... and test.
		c := hconn.Bootstrap(context.Background())
		require.True(t, c.IsSame(ec),
			"should bootstrap when cached client resolves to error")
	})
}
