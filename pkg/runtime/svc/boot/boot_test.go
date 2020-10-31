package boot_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	mock_boot "github.com/wetware/ww/internal/test/mock/pkg/boot"
	mock_vendor "github.com/wetware/ww/internal/test/mock/vendor"
	testutil "github.com/wetware/ww/internal/test/util"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/boot"
	"github.com/wetware/ww/pkg/internal/p2p"
	boot_service "github.com/wetware/ww/pkg/runtime/svc/boot"
	neighborhood_service "github.com/wetware/ww/pkg/runtime/svc/neighborhood"
)

func TestBootstrapperLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	b, err := boot_service.New(boot_service.Config{
		Log:      logger,
		Host:     h,
		Strategy: boot.StaticAddrs{},
	}).Factory.NewService()
	require.NoError(t, err)

	require.Contains(t, b.Loggable(), "service")
	assert.Equal(t, b.Loggable()["service"], "boot")
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

	b, err := boot_service.New(boot_service.Config{
		Log:      logger,
		Host:     h,
		Strategy: s,
	}).Factory.NewService()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, b.Start(ctx))
	defer func() {
		require.NoError(t, b.Stop(ctx))
	}()

	sub, err := bus.Subscribe(new(boot_service.EvtPeerDiscovered))
	require.NoError(t, err)
	defer sub.Close()

	e, err := bus.Emitter(new(neighborhood_service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer e.Close()

	t.Run("Non-Orphaned", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*50)
		defer cancel()

		require.NoError(t, e.Emit(neighborhood_service.EvtNeighborhoodChanged{
			To: neighborhood_service.PhasePartial,
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

		require.NoError(t, e.Emit(neighborhood_service.EvtNeighborhoodChanged{
			To: neighborhood_service.PhaseOrphaned,
		}))

		select {
		case v := <-sub.Out():
			assert.Equal(t, boot_service.EvtPeerDiscovered(info), v.(boot_service.EvtPeerDiscovered))
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})
}

func newMockHost(ctrl *gomock.Controller, bus event.Bus) *mock_vendor.MockHost {
	h := mock_vendor.NewMockHost(ctrl)
	h.EXPECT().
		EventBus().
		Return(bus).
		AnyTimes()

	h.EXPECT().
		ID().
		Return(testutil.RandID()).
		AnyTimes()

	return h
}

// netReady emits p2p.EvtNetworkReady
func netReady(bus event.Bus) error {
	e, err := bus.Emitter(new(p2p.EvtNetworkReady), eventbus.Stateful)
	if err != nil {
		return err
	}

	return e.Emit(p2p.EvtNetworkReady{})
}
