package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventbus "github.com/libp2p/go-eventbus"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	"github.com/wetware/ww/pkg/runtime/service"
)

func TestTicker(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()

	tk, err := service.Ticker(logger, bus, time.Millisecond*10).Service()
	require.NoError(t, err)

	sub, err := bus.Subscribe(new(service.EvtTimestep))
	require.NoError(t, err)

	require.NoError(t, tk.Start(ctx))
	defer func() {
		require.NoError(t, tk.Stop(ctx))
	}()

	// wait until _after_ the 10th tick, but _before_ the 11th.
	done := time.After(time.Millisecond * 105)

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
