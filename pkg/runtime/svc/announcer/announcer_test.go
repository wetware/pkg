package announcer_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	mock_cluster "github.com/wetware/ww/internal/test/mock/pkg/cluster"
	mock_vendor "github.com/wetware/ww/internal/test/mock/vendor"
	testutil "github.com/wetware/ww/internal/test/util"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/wetware/ww/pkg/internal/p2p"
	announcer_service "github.com/wetware/ww/pkg/runtime/svc/announcer"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
)

func TestAnnouncerLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := eventbus.NewBus()

	a, err := announcer_service.New(announcer_service.Config{
		Log:       mock_ww.NewMockLogger(ctrl),
		Host:      newMockHost(ctrl, bus),
		Announcer: mock_cluster.NewMockAnnouncer(ctrl),
		TTL:       time.Second,
	}).Factory.NewService()
	require.NoError(t, err)

	require.Contains(t, a.Loggable(), "service")
	assert.Equal(t, a.Loggable()["service"], "announcer")
	assert.Equal(t, a.Loggable()["ttl"], time.Second)
}

func TestAnnouner(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	bus := eventbus.NewBus()
	ma := mock_cluster.NewMockAnnouncer(ctrl)

	a, err := announcer_service.New(announcer_service.Config{
		Log:       mock_ww.NewMockLogger(ctrl),
		Host:      newMockHost(ctrl, bus),
		Announcer: ma,
		TTL:       time.Second,
	}).Factory.NewService()
	require.NoError(t, err)

	// the service publishes a heartbeat event when it starts
	ma.EXPECT().
		Announce(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, a.Start(ctx))
	defer func() {
		require.NoError(t, a.Stop(ctx))
	}()

	e, err := bus.Emitter(new(tick_service.EvtTimestep))
	require.NoError(t, err)

	t.Run("Announce", func(t *testing.T) {
		ma.EXPECT().
			Announce(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		require.NoError(t, e.Emit(tick_service.EvtTimestep{
			Delta: time.Second * 10,
		}))

		select {
		case <-time.After(time.Millisecond):
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Announce in progress", func(t *testing.T) {
		ma.EXPECT().
			Announce(gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, time.Duration) error {
				// Simulate a slow call to announcer.Announce().
				// While this call blocks, we expect incoming
				// announcement triggers to be dropped.
				time.Sleep(time.Millisecond * 10)
				return nil
			}).
			Times(1)

		for _, ev := range []tick_service.EvtTimestep{{
			Delta: time.Second * 10,
		}, {
			Delta: time.Second * 20,
		}, {
			Delta: time.Second * 30,
		}, {
			Delta: time.Second * 40,
		}} {
			require.NoError(t, e.Emit(ev))
		}

		select {
		case <-time.After(time.Millisecond):
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
