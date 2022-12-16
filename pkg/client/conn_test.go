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

func TestHostConn_Bootstrap(t *testing.T) {
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

func TestHostConn_Close(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := capnp.NewClient(&mockClientHook{})

	conn := mock_client.NewMockConn(ctrl)
	conn.EXPECT().
		Bootstrap(gomock.Any()).
		Return(c).
		Times(1)
	conn.EXPECT().
		Close().
		Return(nil).
		Times(1)

	hconn := &client.HostConn{
		Conn: conn,
	}

	// call once to cache the error client...
	_ = hconn.Bootstrap(context.Background())

	require.NoError(t, hconn.Close(), "should close without error")
	require.Panics(t, func() {
		c.AddRef().Release()
	}, "should release bootstrap client")
}

type mockClientHook struct{}

func (mockClientHook) Send(context.Context, capnp.Send) (*capnp.Answer, capnp.ReleaseFunc) {
	panic("NOT IMPLEMENTED")
}
func (mockClientHook) Recv(context.Context, capnp.Recv) capnp.PipelineCaller {
	panic("NOT IMPLEMENTED")
}
func (mockClientHook) Brand() capnp.Brand { panic("NOT IMPLEMENTED") }
func (mockClientHook) Shutdown()          {}
