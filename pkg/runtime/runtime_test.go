package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-eventbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wetware/ww/pkg/runtime"
	"go.uber.org/fx"
)

func TestRuntime(t *testing.T) {
	t.Run("Succeed", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		var s mockService
		b := runtime.Bundle(provide(&s))
		bus := eventbus.NewBus()

		app := fx.New(fx.NopLogger, fx.Invoke(func(lx fx.Lifecycle) error {
			return runtime.Register(bus, b, lx)
		}))

		require.NoError(t, app.Start(ctx))
		require.NoError(t, app.Stop(ctx))

		assert.True(t, s.start)
		assert.True(t, s.stop)
	})
}

func provide(s runtime.Service) providerFunc {
	return func() (runtime.Service, error) {
		return s, nil
	}
}

type mockService struct {
	start, stop       bool
	startErr, stopErr error
}

func (s *mockService) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"service": "mockService",
	}
}

func (s *mockService) Start(context.Context) error {
	s.start = true
	return s.startErr
}

func (s *mockService) Stop(context.Context) error {
	s.stop = true
	return s.stopErr
}

type providerFunc func() (runtime.Service, error)

func (f providerFunc) Service() (runtime.Service, error) {
	return f()
}
