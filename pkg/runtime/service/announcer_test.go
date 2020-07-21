package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testutil "github.com/wetware/ww/pkg/runtime/service/internal/test"
	mock_service "github.com/wetware/ww/pkg/runtime/service/internal/test/mock"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/service"
)

func TestAnnouncerLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)
	p := mock_service.NewMockPublisher(ctrl)

	a, err := service.Announcer(h, p, time.Second).Service()
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
	h := newMockHost(ctrl, bus)
	p := mock_service.NewMockPublisher(ctrl)

	a, err := service.Announcer(h, p, time.Second).Service()
	require.NoError(t, err)

	// the service publishes a heartbeat event when it starts
	p.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil).
		Times(1)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, testutil.NetReady(bus))

	require.NoError(t, a.Start(ctx))
	defer func() {
		require.NoError(t, a.Stop(ctx))
	}()

	e, err := bus.Emitter(new(service.EvtTimestep))
	require.NoError(t, err)

	t.Run("Announce", func(t *testing.T) {
		p.EXPECT().
			Publish(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(1)

		require.NoError(t, e.Emit(service.EvtTimestep{
			Delta: time.Second * 10,
		}))

		select {
		case <-time.After(time.Millisecond):
		case err := <-a.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Announce in progress", func(t *testing.T) {
		p.EXPECT().
			Publish(gomock.Any(), gomock.Any()).
			DoAndReturn(func(context.Context, []byte) error {
				// Simulate a slow call to announcer.Announce().
				// While this call blocks, we expect incoming
				// announcement triggers to be dropped.
				time.Sleep(time.Millisecond * 10)
				return nil
			}).
			Times(1)

		for _, ev := range []service.EvtTimestep{{
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
		case err := <-a.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})
}
