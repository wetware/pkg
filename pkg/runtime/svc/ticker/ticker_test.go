package ticker_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventbus "github.com/libp2p/go-eventbus"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	tick_service "github.com/wetware/ww/pkg/runtime/svc/ticker"
)

func TestLoggable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tk, err := tick_service.New(tick_service.Config{
		Log: mock_ww.NewMockLogger(ctrl),
		Bus: eventbus.NewBus(),
	}).Factory.NewService()
	require.NoError(t, err)

	assert.Equal(t, "ticker", tk.Loggable()["service"])
	assert.Equal(t, time.Millisecond*100, tk.Loggable()["timestep"])
}

func TestTicker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()

	tk, err := tick_service.New(tick_service.Config{
		Log: logger,
		Bus: bus,
	}).Factory.NewService()
	require.NoError(t, err)

	sub, err := bus.Subscribe(new(tick_service.EvtTimestep))
	require.NoError(t, err)

	require.NoError(t, tk.Start(ctx))
	defer func() {
		require.NoError(t, tk.Stop(ctx))
	}()

	// wait until _after_ the 10th tick, but _before_ the 11th.
	done := time.After(time.Millisecond * 1050)

	var ticks int
	for {
		select {
		case <-sub.Out(): // TODO:  test contents of EvtTimestep
			ticks++
		case <-done:
			assert.Equal(t, 10, ticks)
			return
		}
	}
}
