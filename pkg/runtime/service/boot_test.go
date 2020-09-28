package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	mock_boot "github.com/wetware/ww/internal/test/mock/pkg/boot"
	testutil "github.com/wetware/ww/internal/test/util"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/runtime/service"
)

func TestBootstrapperLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	n, err := service.Bootstrap(logger, h, boot.StaticAddrs{}).Service()
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

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	s := mock_boot.NewMockStrategy(ctrl)

	b, err := service.Bootstrap(logger, h, s).Service()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

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
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})
}
