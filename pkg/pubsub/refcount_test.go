package pubsub_test

import (
	"context"
	"testing"

	capnp "capnproto.org/go/capnp/v3"
	"github.com/golang/mock/gomock"
	logtest "github.com/lthibault/log/test"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/pubsub"
	"golang.org/x/sync/errgroup"
)

func TestRefCount(t *testing.T) {
	t.Parallel()

	/*
		Look for concurrency errors in the refcounting logic by
		performing a large number of concurrent join/leave calls.
	*/

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use the logger to detect failed refcounts.  These will be
	// logged by the Topic server at FATAL level.
	logger := logtest.NewMockLogger(ctrl)
	logger.EXPECT().
		WithField("topic", gomock.Any()).
		AnyTimes()

	ps, release := newGossipSub(ctx)
	defer release()

	r := &pubsub.Server{
		Log:         logger,
		TopicJoiner: ps,
	}

	joiner := r.PubSub()
	defer joiner.Release()

	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 1024; i++ {
		g.Go(stressJoinLeave(ctx, joiner))
	}

	err := g.Wait()
	require.NoError(t, err, "should resolve topic")
}

func stressJoinLeave(ctx context.Context, joiner pubsub.Router) func() error {
	run := func() error {
		topic, release := joiner.Join(ctx, "test")
		defer release()

		return capnp.Client(topic).Resolve(ctx)
	}

	return func() (err error) {
		for i := 0; i < 32; i++ {
			if err = run(); err != nil {
				break
			}
		}

		return
	}
}
