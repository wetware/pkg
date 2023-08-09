package client_test

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wetware/ww/cluster"
	mock_client "github.com/wetware/ww/internal/mock/pkg/client"
	"github.com/wetware/ww/pkg/anchor"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/pubsub"
)

func init() {
	capnp.SetClientLeakFunc(func(msg string) {
		panic(msg)
	})
}

func TestMain(m *testing.M) {
	defer runtime.GC()
	m.Run()
}

func TestNode(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := host.Server{
		ViewProvider:   mockViewProvider{},
		PubSubProvider: mockPubSubProvider{},
		AnchorProvider: mockAnchorProvider{},
	}.Client()
	defer c.Release()

	conn := mock_client.NewMockConn(ctrl)
	conn.EXPECT().
		Bootstrap(gomock.Any()).
		Return(c).
		MinTimes(1)
	conn.EXPECT().
		Close().
		Return(nil).
		Times(1)

	n := &client.Node{Conn: conn}
	defer n.Close()

	assert.Equal(t, "/", n.Path(), "should have root path")

	assert.NoError(t, n.Bootstrap(context.Background()), "should resolve")

	v, release := n.View(context.Background())
	require.NotNil(t, release, "should return release func")
	assert.NotZero(t, v, "should return view")
	defer release()

	topic, release := n.Join(context.Background(), "test")
	require.NotNil(t, release, "should return release func")
	assert.NotZero(t, topic, "should return topic")
	defer release()
}

type mockViewProvider struct{}

func (mockViewProvider) View() cluster.View {
	return cluster.View(capnp.ErrorClient(errors.New("mock")))
}

type mockPubSubProvider struct{}

func (mockPubSubProvider) PubSub() pubsub.Router {
	return pubsub.Router(capnp.ErrorClient(errors.New("mock")))
}

type mockAnchorProvider struct{}

func (mockAnchorProvider) Anchor() anchor.Anchor {
	return anchor.Anchor(capnp.ErrorClient(errors.New("mock")))
}
