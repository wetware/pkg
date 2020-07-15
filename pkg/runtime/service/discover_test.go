package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	testutil "github.com/lthibault/wetware/pkg/runtime/service/internal/test"
	mock_service "github.com/lthibault/wetware/pkg/runtime/service/internal/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/lthibault/wetware/pkg/runtime/service"
)

const ns = "ww.test"

func TestDiscoveryLogFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	d, err := service.Discover(h, ns, nil).Service()
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

	bus := eventbus.NewBus()
	h := newMockHost(ctrl, bus)

	dy := mock_service.NewMockDiscovery(ctrl)

	d, err := service.Discover(h, ns, dy).Service()
	require.NoError(t, err)

	// signal that network is ready; note that this must happen before
	// starting the neighborhood service
	require.NoError(t, testutil.NetReady(bus))

	require.NoError(t, d.Start(ctx))
	defer func() {
		require.NoError(t, d.Stop(ctx))
	}()

	/*
		TODO(testing):  add discover-specific tests here.
	*/
}
