package pubsub_test

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	capnp "capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/rpc"
	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	pscap "github.com/wetware/ww/pkg/pubsub"
)

func TestMain(m *testing.M) {
	capnp.SetClientLeakFunc(func(msg string) {
		fmt.Println(msg)
	})
	defer runtime.GC()

	m.Run()
}

func TestPubSub(t *testing.T) {
	t.Parallel()

	const topic = "test"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	ps := (&pscap.Server{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	const nmsg = 10
	g, ctx := errgroup.WithContext(ctx)

	// belt-and-suspenders; make sure we don't miss a message due
	// to goroutine scheduling
	sync := make(chan struct{})

	// reader
	g.Go(func() error {
		topic, release := ps.Join(ctx, topic)
		defer release()

		sub, cancel := topic.Subscribe(ctx)
		defer cancel()

		// Ready to receive.  In principle, the RPC request may still be
		// in-flight, but this should be good enough.  There aren't really
		// any ordering guarantees.
		//
		// If this test becomes unreliable due to missed messages, we should
		// remove synchronization and test that at least one message makes it
		// through.
		close(sync)

		var i int
		for got := sub.Next(); got != nil; got = sub.Next() {
			if string(got) != "test" {
				return fmt.Errorf("reader: unexpected message: %s", got)
			}

			i++
			t.Logf("got message %d of %d", i, nmsg)

			if i == nmsg {
				break
			}
		}

		return sub.Err()
	})

	// writer
	g.Go(func() error {
		<-sync

		topic, release := ps.Join(ctx, topic)
		defer release()

		// Publish 10 messages
		g, ctx := errgroup.WithContext(ctx)
		for i := 0; i < nmsg; i++ {
			g.Go(func() error {
				return topic.Publish(ctx, []byte("test"))
			})
		}

		return annotate("writer", g.Wait())
	})

	assert.NoError(t, g.Wait())
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

	ps := (&pscap.Server{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	p0, p1 := net.Pipe()
	c0 := rpc.NewConn(rpc.NewStreamTransport(p0), &rpc.Options{
		BootstrapClient: capnp.Client(ps),
	})
	defer c0.Close()

	c1 := rpc.NewConn(rpc.NewStreamTransport(p1), nil)
	defer c1.Close()

	joiner := pscap.Router(c1.Bootstrap(ctx))
	defer joiner.Release()

	topic, release := joiner.Join(ctx, "test")
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

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	/*
		This is a regression test that ensures pubsub messages are
		copied prior to releasing their underlying Cap'n Proto segments.

		Starting with Cap'n Proto v3.0.0-alpha.10, RPC messages and their
		segments are pooled and zeroed between use.  This would manifest
		as messages containing only null bytes.
	*/

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gs, release := newGossipSub(ctx)
	defer release()

	ps := (&pscap.Server{TopicJoiner: gs}).PubSub()
	defer ps.Release()

	p0, p1 := net.Pipe()
	c0 := rpc.NewConn(rpc.NewStreamTransport(p0), &rpc.Options{
		BootstrapClient: capnp.Client(ps),
	})
	defer c0.Close()

	c1 := rpc.NewConn(rpc.NewStreamTransport(p1), nil)
	defer c1.Close()

	joiner := pscap.Router(c1.Bootstrap(ctx))
	defer joiner.Release()

	topic, release := joiner.Join(ctx, "test")
	defer release()

	sub, release := topic.Subscribe(ctx)
	defer release()

	cherr := make(chan error, 1)
	go func() {
		defer close(cherr)
		cherr <- topic.Publish(ctx, []byte("test"))
	}()

	require.NoError(t, <-cherr,
		"must publish test message before testing payload")

	assert.Equal(t, "test", string(sub.Next()),
		"should copy message payload before segment is zeroed")
}

func annotate(prefix string, err error) error {
	if err != nil {
		err = fmt.Errorf("%s: %w", prefix, err)
	}

	return err
}

func newGossipSub(ctx context.Context) (*pubsub.PubSub, func()) {
	h := newTestHost()

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	return ps, func() { h.Close() }
}

func newTestHost() host.Host {
	h, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.NoTransports,
		libp2p.Transport(inproc.New()),
		libp2p.ListenAddrStrings("/inproc/~"))
	if err != nil {
		panic(err)
	}

	return h
}
