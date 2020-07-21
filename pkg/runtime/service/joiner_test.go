package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testutil "github.com/wetware/ww/pkg/runtime/service/internal/test"
	mock_service "github.com/wetware/ww/pkg/runtime/service/internal/test/mock"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/wetware/ww/pkg/runtime"
	"github.com/wetware/ww/pkg/runtime/service"
)

func TestJoiner(t *testing.T) {
	t.Run("EvtPeerDiscovered triggers connection", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		bus := eventbus.NewBus()
		h := newMockHost(ctrl, bus)

		j, err := service.Joiner(h).Service()
		require.NoError(t, err)

		// signal that the network is ready; note that this must happen before the
		// joiner service is started.
		require.NoError(t, testutil.NetReady(bus))

		// start the service
		require.NoError(t, j.Start(ctx))
		defer func() {
			require.NoError(t, j.Stop(ctx))
		}()

		called := make(chan struct{})
		h.EXPECT().
			Connect(gomock.Any(), gomock.Any()). // TODO:  check value of the multiaddr
			Return(nil).
			Do(func(context.Context, peer.AddrInfo) {
				close(called)
			}).
			Times(1)

		e, err := bus.Emitter(new(service.EvtPeerDiscovered))
		require.NoError(t, err)
		defer e.Close()

		err = e.Emit(service.EvtPeerDiscovered{})
		require.NoError(t, err)

		select {
		case <-called:
		case err := <-j.(runtime.ErrorStreamer).Errors():
			t.Error(err)
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	t.Run("Connection errors reported", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		bus := eventbus.NewBus()
		h := newMockHost(ctrl, bus)

		j, err := service.Joiner(h).Service()
		require.NoError(t, err)

		// signal that the network is ready; note that this must happen before the
		// joiner service is started.
		require.NoError(t, testutil.NetReady(bus))

		// start the service
		require.NoError(t, j.Start(ctx))
		defer func() {
			require.NoError(t, j.Stop(ctx))
		}()

		h.EXPECT().
			Connect(gomock.Any(), gomock.Any()). // TODO:  check value of the multiaddr
			Return(errors.New("[ TEST ERROR ]")).
			Times(1)

		e, err := bus.Emitter(new(service.EvtPeerDiscovered))
		require.NoError(t, err)
		defer e.Close()

		err = e.Emit(service.EvtPeerDiscovered{})
		require.NoError(t, err)

		select {
		case err := <-j.(runtime.ErrorStreamer).Errors():
			assert.EqualError(t, err, "[ TEST ERROR ]")
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	})

	/*
		TODO(enhancement):  test behavior when multiple EvtPeerDiscovered events are
							emitted in quick succession.

		Currently the strategy is to drop all events while a connection attempt is in
		progress.  We may decide to fine-tune this strategy at a later date, at which
		point it should be tested.
	*/
}

func newMockHost(ctrl *gomock.Controller, bus event.Bus) *mock_service.MockHost {
	h := mock_service.NewMockHost(ctrl)
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
