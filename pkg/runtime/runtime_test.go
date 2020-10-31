package runtime_test

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/golang/mock/gomock"
// 	"github.com/lthibault/log"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
// 	ww "github.com/wetware/ww/pkg"
// 	"github.com/wetware/ww/pkg/runtime"
// 	"go.uber.org/fx"
// )

// func TestRuntime(t *testing.T) {
// 	t.Run("Succeed", func(t *testing.T) {
// 		t.Parallel()

// 		ctrl := gomock.NewController(t)
// 		defer ctrl.Finish()

// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 		defer cancel()

// 		var s mockService
// 		b := runtime.Bundle(provide(&s))

// 		// Ensure services are automatically logged
// 		logger := mock_ww.NewMockLogger(ctrl)

// 		// We expect one call to the logger at start.  There is no corresponding
// 		// call at stop because the service finishes gracefully.
// 		logger.EXPECT().
// 			With(gomock.Any()).
// 			DoAndReturn(func(log.Loggable) ww.Logger { return logger }).
// 			Times(1)
// 		logger.EXPECT().
// 			Debug("service started").
// 			Times(1)

// 		app := fx.New(fx.NopLogger, fx.Invoke(func(lx fx.Lifecycle) error {
// 			return runtime.Register(logger, b, lx)
// 		}))

// 		require.NoError(t, app.Start(ctx))
// 		require.NoError(t, app.Stop(ctx))

// 		assert.True(t, s.start)
// 		assert.True(t, s.stop)
// 	})
// }

// func provide(s runtime.Service) providerFunc {
// 	return func() (runtime.Service, error) {
// 		return s, nil
// 	}
// }

// type mockService struct {
// 	start, stop       bool
// 	startErr, stopErr error
// }

// func (s *mockService) Loggable() map[string]interface{} {
// 	return map[string]interface{}{
// 		"service":  "mockService",
// 		"startErr": s.startErr,
// 		"stopErr":  s.stopErr,
// 	}
// }

// func (s *mockService) Start(context.Context) error {
// 	s.start = true
// 	return s.startErr
// }

// func (s *mockService) Stop(context.Context) error {
// 	s.stop = true
// 	return s.stopErr
// }

// type providerFunc func() (runtime.Service, error)

// func (f providerFunc) Service() (runtime.Service, error) {
// 	return f()
// }
