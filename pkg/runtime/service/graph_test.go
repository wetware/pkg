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

func TestGraphLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	g, err := service.Graph(logger, h).Service()
	require.NoError(t, err)

	require.Contains(t, g.Loggable(), "service")
	assert.Equal(t, g.Loggable()["service"], "graph")
}

func TestGraph(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	g, err := service.Graph(logger, h).Service()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, g.Start(ctx))
	defer func() {
		require.NoError(t, g.Stop(ctx))
	}()

	e, err := bus.Emitter(new(service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer e.Close()

	t.Run("Boot", func(t *testing.T) {
		boot, err := bus.Subscribe(new(service.EvtBootRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhaseOrphaned,
		}))

		select {
		case <-boot.Out():
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Graft", func(t *testing.T) {
		graft, err := bus.Subscribe(new(service.EvtGraftRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhasePartial,
		}))

		select {
		case <-graft.Out():
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Prune", func(t *testing.T) {
		prune, err := bus.Subscribe(new(service.EvtPruneRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(service.EvtNeighborhoodChanged{
			To: service.PhaseOverloaded,
		}))

		select {
		case <-prune.Out():
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

}
