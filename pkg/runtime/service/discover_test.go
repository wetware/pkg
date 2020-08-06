package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_vendor "github.com/wetware/ww/internal/test/mock/vendor"

	eventbus "github.com/libp2p/go-eventbus"
	"github.com/wetware/ww/pkg/runtime/service"
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

	dy := mock_vendor.NewMockDiscovery(ctrl)

	d, err := service.Discover(h, ns, dy).Service()
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
