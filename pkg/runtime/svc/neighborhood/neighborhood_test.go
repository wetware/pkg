package neighborhood_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testutil "github.com/wetware/ww/internal/test/util"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/wetware/ww/pkg/internal/p2p"
	neighborhood_service "github.com/wetware/ww/pkg/runtime/svc/neighborhood"
)

const (
	kmin = 3
	kmax = 4
)

func TestNeighborhoodLogFields(t *testing.T) {
	t.Parallel()
	bus := eventbus.NewBus()

	n, err := neighborhood_service.New(neighborhood_service.Config{
		Bus:  bus,
		KMin: kmin,
		KMax: kmax,
	}).Factory.NewService()
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

	n, err := neighborhood_service.New(neighborhood_service.Config{
		Bus:  bus,
		KMin: kmin,
		KMax: kmax,
	}).Factory.NewService()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, n.Start(ctx))
	defer func() {
		require.NoError(t, n.Stop(ctx))
	}()

	sub, err := bus.Subscribe(new(neighborhood_service.EvtNeighborhoodChanged))
	require.NoError(t, err)
	defer sub.Close()

	// EvtNeighborhoodChanged is a STATEFUL subscriber.  Start by checking that its
	// initial state is expected.
	select {
	case v := <-sub.Out():
		ev := v.(neighborhood_service.EvtNeighborhoodChanged)
		require.Zero(t, ev.K)
		require.Zero(t, ev.From)
		require.Zero(t, ev.To)
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
			From, To neighborhood_service.Phase
		}{{
			From: neighborhood_service.PhaseOrphaned,
			K:    1, // 0 => 1
			To:   neighborhood_service.PhasePartial,
		}, {
			From: neighborhood_service.PhasePartial,
			K:    2, // 1 => 2
			To:   neighborhood_service.PhasePartial,
		}, {
			From: neighborhood_service.PhasePartial,
			K:    3, // 2 => 3
			To:   neighborhood_service.PhaseComplete,
		}, {
			From: neighborhood_service.PhaseComplete,
			K:    4, // 3 => 4
			To:   neighborhood_service.PhaseComplete,
		}, {
			From: neighborhood_service.PhaseComplete,
			K:    5, // 4 => 5
			To:   neighborhood_service.PhaseOverloaded,
		}} {
			err = e.Emit(evtPeerConnectednessChanged(pids[i], network.Connected))
			require.NoError(t, err)

			select {
			case v := <-sub.Out():
				ev := v.(neighborhood_service.EvtNeighborhoodChanged)
				assert.Equal(t, tC.K, ev.K)
				assert.Equal(t, tC.From, ev.From,
					"expected previous phase %s, got %s", tC.From, ev.From)
				assert.Equal(t, tC.To, ev.To,
					"expected current phase %s, got %s", tC.To, ev.To)
			case <-ctx.Done():
				t.Error(ctx.Err())
			}
		}
	})

	t.Run("Peers disconnecting", func(t *testing.T) {
		for i, tC := range []struct {
			K        int
			From, To neighborhood_service.Phase
		}{{
			From: neighborhood_service.PhaseOverloaded,
			K:    4,
			To:   neighborhood_service.PhaseComplete,
		}, {
			From: neighborhood_service.PhaseComplete,
			K:    3,
			To:   neighborhood_service.PhaseComplete,
		}, {
			From: neighborhood_service.PhaseComplete,
			K:    2,
			To:   neighborhood_service.PhasePartial,
		}, {
			From: neighborhood_service.PhasePartial,
			K:    1,
			To:   neighborhood_service.PhasePartial,
		}, {
			From: neighborhood_service.PhasePartial,
			K:    0,
			To:   neighborhood_service.PhaseOrphaned,
		}} {
			err = e.Emit(evtPeerConnectednessChanged(pids[i], network.NotConnected))
			require.NoError(t, err)

			select {
			case v := <-sub.Out():
				ev := v.(neighborhood_service.EvtNeighborhoodChanged)
				assert.Equal(t, tC.K, ev.K)
				assert.Equal(t, tC.From, ev.From,
					"expected previous phase %s, got %s", tC.From, ev.From)
				assert.Equal(t, tC.To, ev.To,
					"expected current phase %s, got %s", tC.To, ev.To)
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

// netReady emits p2p.EvtNetworkReady
func netReady(bus event.Bus) error {
	e, err := bus.Emitter(new(p2p.EvtNetworkReady), eventbus.Stateful)
	if err != nil {
		return err
	}

	return e.Emit(p2p.EvtNetworkReady{})
}
