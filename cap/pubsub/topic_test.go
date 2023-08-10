package pubsub_test

import (
	"context"
	"testing"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/wetware/pkg/api/pubsub"
	"github.com/wetware/pkg/cap/pubsub"
	test_pubsub "github.com/wetware/pkg/cap/pubsub/test"
)

func TestPublish(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := test_pubsub.NewMockTopicServer(ctrl)
	server.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, call api.Topic_publish) {
			b, err := call.Args().Msg()
			assert.NoError(t, err, "should have message argument")
			assert.Equal(t, "hello, world!", string(b),
				"argument should match input")
		}).
		Return(nil).
		Times(1)

	topic := pubsub.NewTopic(server)
	defer topic.Release()

	err := topic.Publish(context.Background(), []byte("hello, world!"))
	require.NoError(t, err, "should succeed")
	require.NoError(t, capnp.Client(topic).WaitStreaming())
}
