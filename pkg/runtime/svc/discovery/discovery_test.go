package discover_test

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
	discovery_service "github.com/wetware/ww/pkg/runtime/svc/discovery"
)

const ns = "ww.test"

func TestDiscoveryLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	d, err := discovery_service.New(discovery_service.Config{
		Log:       logger,
		Host:      h,
		Namespace: ns,
		Discovery: nil,
	}).Factory.NewService()
	require.NoError(t, err)

	require.Contains(t, d.Loggable(), "service")
	assert.Equal(t, d.Loggable()["service"], "discover")
	assert.Equal(t, d.Loggable()["ns"], ns)
}

func TestDiscovery(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	logger := mock_ww.NewMockLogger(ctrl)
	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	dy := mock_vendor.NewMockDiscovery(ctrl)

	d, err := discovery_service.New(discovery_service.Config{
		Log:       logger,
		Host:      h,
		Namespace: ns,
		Discovery: dy,
	}).Factory.NewService()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, netReady(bus))

	require.NoError(t, d.Start(ctx))
	defer func() {
		require.NoError(t, d.Stop(ctx))
	}()

	/*
		TODO(testing):  add discover-specific tests here.
	*/
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
