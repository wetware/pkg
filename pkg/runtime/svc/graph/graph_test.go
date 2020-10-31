package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	mock_vendor "github.com/wetware/ww/internal/test/mock/vendor"
	testutil "github.com/wetware/ww/internal/test/util"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/wetware/ww/pkg/internal/p2p"
	graph_service "github.com/wetware/ww/pkg/runtime/svc/graph"
	neighborhood_service "github.com/wetware/ww/pkg/runtime/svc/neighborhood"
)

func TestGraphLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	g, err := graph_service.New(graph_service.Config{
		Log:  logger,
		Host: h,
	}).Factory.NewService()
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

	g, err := graph_service.New(graph_service.Config{
		Log:  logger,
		Host: h,
	}).Factory.NewService()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, g.Start(ctx))
	defer func() {
		require.NoError(t, g.Stop(ctx))
	}()

	e, err := bus.Emitter(new(neighborhood_service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer e.Close()

	t.Run("Boot", func(t *testing.T) {
		boot, err := bus.Subscribe(new(graph_service.EvtBootRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(neighborhood_service.EvtNeighborhoodChanged{
			To: neighborhood_service.PhaseOrphaned,
		}))

		select {
		case <-boot.Out():
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Graft", func(t *testing.T) {
		graft, err := bus.Subscribe(new(graph_service.EvtGraftRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(neighborhood_service.EvtNeighborhoodChanged{
			To: neighborhood_service.PhasePartial,
		}))

		select {
		case <-graft.Out():
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Prune", func(t *testing.T) {
		prune, err := bus.Subscribe(new(graph_service.EvtPruneRequested))
		require.NoError(t, err)

		require.NoError(t, e.Emit(neighborhood_service.EvtNeighborhoodChanged{
			To: neighborhood_service.PhaseOverloaded,
		}))

		select {
		case <-prune.Out():
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
