package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/lthibault/wetware/pkg/runtime"
	"github.com/lthibault/wetware/pkg/runtime/service"
)

func TestTicker(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	bus := eventbus.NewBus()

	tk, err := service.Ticker(bus, time.Millisecond*10).Service()
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
		case err := <-tk.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-done:
			assert.Equal(t, 10, ticks)
			return
		}
	}
}
