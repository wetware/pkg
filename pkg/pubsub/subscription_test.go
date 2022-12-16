package pubsub_test

import (
	"context"
	"testing"
	"time"

	"capnproto.org/go/capnp/v3/exc"
	"github.com/stretchr/testify/assert"
	"github.com/wetware/ww/pkg/pubsub"
)

func TestNullSubscription(t *testing.T) {
	t.Parallel()

	topic, release := pubsub.Topic{}.Subscribe(context.Background())
	defer release()

	assert.Nil(t, topic.Next(), "should be exhausted")

	var target exc.Exception
	assert.ErrorAs(t, topic.Err(), &target,
		"should report null-client exception")
	assert.Equal(t, exc.Failed, target.Type,
		"should return 'failed' exception type")
}

func TestSubscribe_cancel(t *testing.T) {
	t.Parallel()

	/*
		Test that releasing a subscription causes the iterator to
		unblock.
	*/

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	router := (&pubsub.Server{TopicJoiner: gs}).PubSub()
	defer router.Release()

	topic, release := router.Join(ctx, "test")
	defer release()

	sub, release := topic.Subscribe(ctx)
	defer release()

	release()
	assert.Eventually(t, func() bool {
		select {
		case <-sub.Future.Done():
			return true
		default:
			return false
		}
	}, time.Millisecond*100, time.Millisecond*10,
		"should eventually abort iteration")
}
