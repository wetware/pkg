package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	testutil "github.com/lthibault/wetware/pkg/runtime/service/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/lthibault/wetware/pkg/runtime"
	"github.com/lthibault/wetware/pkg/runtime/service"
)

const (
	kmin = 3
	kmax = 4
)

func TestNeighborhoodLogFields(t *testing.T) {
	t.Parallel()
	bus := eventbus.NewBus()

	n, err := service.Neighborhood(bus, kmin, kmax).Service()
	require.NoError(t, err)

	require.Contains(t, n.Loggable(), "service")
	assert.Equal(t, n.Loggable()["service"], "neighborhood")
}

func TestNeighborhoodPhase(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	bus := eventbus.NewBus()

	n, err := service.Neighborhood(bus, kmin, kmax).Service()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, testutil.NetReady(bus))

	require.NoError(t, n.Start(ctx))
	defer func() {
		require.NoError(t, n.Stop(ctx))
	}()

	sub, err := bus.Subscribe(new(service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer sub.Close()

	// EvtNeighborhoodChanged is a STATEFUL subscriber.  Start by checking that its
	// initial state is expected.
	select {
	case v := <-sub.Out():
		ev := v.(service.EvtNeighborhoodChanged)
		require.Zero(t, ev.K)
		require.Zero(t, ev.From)
		require.Zero(t, ev.To)
	case err := <-n.(runtime.ErrorStreamer).Errors():
		t.Error(err)
	case <-ctx.Done():
		t.Error(ctx.Err())
	}

	// Now signal changes in peer connectivity and test that the neighborhood state
	// remains consistent.
	e, err := bus.Emitter(new(event.EvtPeerConnectednessChanged))
	require.NoError(t, err)

	pids := make([]peer.ID, 5)
	for i := range pids {
		pids[i] = testutil.RandID()
	}

	t.Run("Peers connecting", func(t *testing.T) {
		for i, tC := range []struct {
			K        int
			From, To service.Phase
		}{{
			From: service.PhaseOrphaned,
			K:    1, // 0 => 1
			To:   service.PhasePartial,
		}, {
			From: service.PhasePartial,
			K:    2, // 1 => 2
			To:   service.PhasePartial,
		}, {
			From: service.PhasePartial,
			K:    3, // 2 => 3
			To:   service.PhaseComplete,
		}, {
			From: service.PhaseComplete,
			K:    4, // 3 => 4
			To:   service.PhaseComplete,
		}, {
			From: service.PhaseComplete,
			K:    5, // 4 => 5
			To:   service.PhaseOverloaded,
		}} {
			err = e.Emit(evtPeerConnectednessChanged(pids[i], network.Connected))
			require.NoError(t, err)

			select {
			case v := <-sub.Out():
				ev := v.(service.EvtNeighborhoodChanged)
				assert.Equal(t, tC.K, ev.K)
				assert.Equal(t, tC.From, ev.From,
					"expected previous phase %s, got %s", tC.From, ev.From)
				assert.Equal(t, tC.To, ev.To,
					"expected current phase %s, got %s", tC.To, ev.To)
			case err := <-n.(runtime.ErrorStreamer).Errors():
				t.Error(err)
			case <-ctx.Done():
				t.Error(ctx.Err())
			}
		}
	})

	t.Run("Peers disconnecting", func(t *testing.T) {
		for i, tC := range []struct {
			K        int
			From, To service.Phase
		}{{
			From: service.PhaseOverloaded,
			K:    4,
			To:   service.PhaseComplete,
		}, {
			From: service.PhaseComplete,
			K:    3,
			To:   service.PhaseComplete,
		}, {
			From: service.PhaseComplete,
			K:    2,
			To:   service.PhasePartial,
		}, {
			From: service.PhasePartial,
			K:    1,
			To:   service.PhasePartial,
		}, {
			From: service.PhasePartial,
			K:    0,
			To:   service.PhaseOrphaned,
		}} {
			err = e.Emit(evtPeerConnectednessChanged(pids[i], network.NotConnected))
			require.NoError(t, err)

			select {
			case v := <-sub.Out():
				ev := v.(service.EvtNeighborhoodChanged)
				assert.Equal(t, tC.K, ev.K)
				assert.Equal(t, tC.From, ev.From,
					"expected previous phase %s, got %s", tC.From, ev.From)
				assert.Equal(t, tC.To, ev.To,
					"expected current phase %s, got %s", tC.To, ev.To)
			case err := <-n.(runtime.ErrorStreamer).Errors():
				t.Error(err)
			case <-ctx.Done():
				t.Error(ctx.Err())
			}
		}
	})
}

func evtPeerConnectednessChanged(id peer.ID, c network.Connectedness) event.EvtPeerConnectednessChanged {
	return event.EvtPeerConnectednessChanged{
		Peer:          id,
		Connectedness: c,
	}
}
