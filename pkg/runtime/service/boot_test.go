package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testutil "github.com/wetware/ww/pkg/runtime/service/internal/test"
	mock_service "github.com/wetware/ww/pkg/runtime/service/internal/test/mock"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/service"
)

func TestBootstrapperLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	n, err := service.Bootstrap(h, boot.StaticAddrs{}).Service()
	require.NoError(t, err)

	require.Contains(t, n.Loggable(), "service")
	assert.Equal(t, n.Loggable()["service"], "boot")
}

func TestBootstrapper(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	s := mock_service.NewMockBootStrategy(ctrl)

	b, err := service.Bootstrap(h, s).Service()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, testutil.NetReady(bus))

	require.NoError(t, b.Start(ctx))
	defer func() {
		require.NoError(t, b.Stop(ctx))
	}()

	sub, err := bus.Subscribe(new(service.EvtPeerDiscovered))
	require.NoError(t, err)
	defer sub.Close()

	e, err := bus.Emitter(new(service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer e.Close()

	t.Run("Non-Orphaned", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*50)
		defer cancel()

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhasePartial,
		}))

		select {
		case v := <-sub.Out():
			t.Errorf("non-orphaned node initiated peer discovery:  %v", v)
		case err := <-b.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-ctx.Done():
		}
	})

	t.Run("Orphaned", func(t *testing.T) {
		info := peer.AddrInfo{ID: testutil.RandID()}
		ch := make(chan peer.AddrInfo, 1)
		ch <- info
		defer close(ch)

		s.EXPECT().
			DiscoverPeers(gomock.Any(), gomock.Any()).
			Return((<-chan peer.AddrInfo)(ch), nil).
			Times(1)

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhaseOrphaned,
		}))

		select {
		case v := <-sub.Out():
			assert.Equal(t, service.EvtPeerDiscovered(info), v.(service.EvtPeerDiscovered))
		case err := <-b.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Orphaned error", func(t *testing.T) {
		info := peer.AddrInfo{ID: testutil.RandID()}
		ch := make(chan peer.AddrInfo, 1)
		ch <- info
		defer close(ch)

		s.EXPECT().
			DiscoverPeers(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("[ TEST ERROR ]")).
			Times(1)

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhaseOrphaned,
		}))

		select {
		case v := <-sub.Out():
			t.Errorf("expected error, got %v", v)
		case err := <-b.(runtime.ErrorStreamer).Errors():
			assert.EqualError(t, err, "[ TEST ERROR ]")
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})
}
