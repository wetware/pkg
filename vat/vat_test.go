package vat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	inproc "github.com/lthibault/go-libp2p-inproc-transport"
	"github.com/stretchr/testify/require"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/vat"
)

func TestServe(t *testing.T) {
	t.Parallel()
	t.Helper()

	var addr = make(chan beacon, 1)

	t.Run("serverThread", func(t *testing.T) {
		t.Parallel()

		const d = time.Millisecond * 100
		ctx, cancel := context.WithTimeout(context.Background(), d)
		defer cancel()

		h, err := libp2p.New(
			libp2p.NoTransports,
			libp2p.NoListenAddrs,
			libp2p.Transport(inproc.New()),
			libp2p.ListenAddrStrings("/inproc/~"))
		require.NoError(t, err)
		defer h.Close()

		addr <- beacon{
			AddrInfo: host.InfoFromHost(h),
			Bus:      h.EventBus(),
		}

		dht, err := vat.NewDHT(ctx, h, "test")
		require.NoError(t, err)
		defer dht.Close()

		err = vat.Config{
			NS:        "test",
			Host:      h,
			Bootstrap: nopDiscovery{},
			Ambient:   nopDiscovery{},
			Auth:      auth.AllowAll,
		}.Serve(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	// wait for the server to come online
	time.Sleep(time.Millisecond * 10)

	t.Run("clientThread", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Due to a limitation in libp2p, we cannot reuse the host.  This is
		// because hosts can't dial to themselves.  At some point, it may be
		// worthwhile to contribute this feature to libp2p.  In the meantime,
		// we'll just start a separate host for the client.
		h, err := libp2p.New(
			libp2p.NoTransports,
			libp2p.NoListenAddrs,
			libp2p.Transport(inproc.New()))
		require.NoError(t, err)
		defer h.Close()

		sess, err := vat.Dialer{
			Host:    h,
			Account: auth.SignerFromHost(h),
		}.DialDiscover(ctx, <-addr, "test")
		defer sess.Release()
		require.NoError(t, err)
		require.NotZero(t, sess)
	})
}

type beacon struct {
	*peer.AddrInfo
	event.Bus
}

func (beacon) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return nopDiscovery{}.Advertise(context.TODO(), "")
}

func (b beacon) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	a := make(announcer, 1)
	defer close(a)

	a <- *b.AddrInfo

	// Subscribe to the server's event bus, so that we know when it's up.
	sub, err := b.Bus.Subscribe(new(auth.Session))
	if err != nil {
		return nil, err
	}
	defer sub.Close()

	// Wait for the server to become ready, then let the client know about
	// it.
	select {
	case _, ok := <-sub.Out():
		if ok {
			return a.FindPeers(ctx, ns, opt...)
		}
		return nil, errors.New("event.Bus closed")

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type announcer chan peer.AddrInfo

func (announcer) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return nopDiscovery{}.Advertise(context.TODO(), "")
}

func (a announcer) FindPeers(ctx context.Context, _ string, _ ...discovery.Option) (<-chan peer.AddrInfo, error) {
	return a, nil
}

type nopDiscovery struct{}

func (nopDiscovery) Advertise(context.Context, string, ...discovery.Option) (time.Duration, error) {
	return peerstore.PermanentAddrTTL, nil
}
func (nopDiscovery) FindPeers(ctx context.Context, _ string, _ ...discovery.Option) (<-chan peer.AddrInfo, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
