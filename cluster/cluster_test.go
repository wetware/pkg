package cluster_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	logtest "github.com/lthibault/log/test"
	"github.com/wetware/pkg/cluster"
	test_cluster "github.com/wetware/pkg/cluster/test"

	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var relayCanceled atomic.Bool

	logger := logtest.NewMockLogger(ctrl)
	logger.EXPECT().
		With(gomock.Any()).
		Return(logger).
		Times(1)

	table := test_cluster.NewMockRoutingTable(ctrl)
	table.EXPECT().
		Advance(gomock.AssignableToTypeOf(time.Time{})).
		AnyTimes()

	topic := test_cluster.NewMockTopic(ctrl)
	topic.EXPECT().
		String().
		Return("casm").
		AnyTimes()
	topic.EXPECT().
		Relay().
		Return(func() { relayCanceled.Store(true) }, nil).
		Times(1)
	sync := make(chan struct{})
	boot := topic.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, []byte, ...pubsub.PubOpt) error {
			close(sync)
			return nil
		}).
		Times(1)
	topic.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		After(boot).
		AnyTimes()

	router := cluster.Router{
		Log:          logger,
		Topic:        topic,
		RoutingTable: table,
	}
	defer router.Stop()

	err := router.Bootstrap(context.Background())
	assert.NoError(t, err, "bootstrap should succeed")

	select {
	case <-sync:
	case <-time.After(time.Second):
		t.Error("Call to Bootstrap() should trigger call to Publish()")
	}

	router.Stop()
	assert.Eventually(t, relayCanceled.Load,
		time.Millisecond*100, time.Millisecond*10,
		"should cancel topic relay")

	err = router.Bootstrap(context.Background())
	assert.ErrorIs(t, err, cluster.ErrClosing,
		"should not bootstrap after router was stopped")
}
