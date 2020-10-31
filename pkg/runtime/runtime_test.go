package runtime_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mock_ww "github.com/wetware/ww/internal/test/mock/pkg"
	mock_runtime "github.com/wetware/ww/internal/test/mock/pkg/runtime"
	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/runtime"
	"go.uber.org/fx/fxtest"
)

var (
	matchLoggable = gomock.AssignableToTypeOf(reflect.TypeOf(new(ww.Loggable)).Elem())
	matchContext  = gomock.AssignableToTypeOf(reflect.TypeOf(new(context.Context)).Elem())

	errTest = errors.New("test")
)

func TestRuntime(t *testing.T) {
	t.Parallel()

	for desc, factory := range map[string]func(*gomock.Controller) testSpec{
		"Nop":                     testNop,
		"Success":                 testSuccess,
		"SuccessWithDependencies": testSuccessWithDependencies,
		"FactoryError":            testFactoryError,
		"MissingEventProvider":    testMissingEventProvider,
		"StartFailure":            testStartFailure,
		"StopFailure":             testStopFailure,
		"UncleanShutdown":         testUncleanShutdown,
	} {
		t.Run(desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tt := factory(ctrl)
			tt.run(t, fxtest.NewLifecycle(t), runtime.Config{
				Log:      tt.log,
				Services: tt.ss,
			})
		})
	}
}

type testSpec struct {
	ss  []runtime.ServiceFactory
	log *mock_ww.MockLogger
	run func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config)
}

func testNop(*gomock.Controller) testSpec {
	return testSpec{
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			if assert.NoError(t, runtime.Start(cfg, lx)) {
				lx.RequireStart().RequireStop()
			}
		},
	}
}

func testSuccess(ctrl *gomock.Controller) testSpec {
	log := mock_ww.NewMockLogger(ctrl)

	log.EXPECT().
		With(matchLoggable).
		Return(log).
		Times(1)

	log.EXPECT().
		Debug("service started").
		Times(1)

	svc := mock_runtime.NewMockService(ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(nil).
		Times(1)

	return testSpec{
		ss:  []runtime.ServiceFactory{factoryFor(svc)},
		log: log,
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			if assert.NoError(t, runtime.Start(cfg, lx)) {
				lx.RequireStart().RequireStop()
			}
		},
	}
}

func testSuccessWithDependencies(ctrl *gomock.Controller) testSpec {
	log := mock_ww.NewMockLogger(ctrl)

	log.EXPECT().
		With(matchLoggable).
		Return(log).
		Times(2)

	log.EXPECT().
		Debug("service started").
		Times(2)

	svc := mock_runtime.NewMockService(ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(2)

	svc.EXPECT().
		Stop(matchContext).
		Return(nil).
		Times(2)

	return testSpec{
		ss: []runtime.ServiceFactory{
			produces(ctrl, svc, struct{}{}),
			consumes(ctrl, svc, struct{}{}),
		},
		log: log,
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			if assert.NoError(t, runtime.Start(cfg, lx)) {
				lx.RequireStart().RequireStop()
			}
		},
	}
}

func testFactoryError(ctrl *gomock.Controller) testSpec {
	return testSpec{
		ss: []runtime.ServiceFactory{
			errFactory(errTest),
			factoryFor(mock_runtime.NewMockService(ctrl)),
		},
		log: mock_ww.NewMockLogger(ctrl),
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			assert.EqualError(t, runtime.Start(cfg, lx), errTest.Error(),
				"error from ServiceFactory.NewService() not reported")
		},
	}
}

func testMissingEventProvider(ctrl *gomock.Controller) testSpec {
	return testSpec{
		ss: []runtime.ServiceFactory{
			consumes(ctrl, mock_runtime.NewMockService(ctrl), struct{}{}),
		},
		log: mock_ww.NewMockLogger(ctrl),
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			assert.EqualError(t,
				runtime.Start(cfg, lx),
				runtime.DependencyError{reflect.TypeOf(struct{}{})}.Error(),
				"error from unresolved dependency not reported")
		},
	}
}

func testStartFailure(ctrl *gomock.Controller) testSpec {
	svc := mock_runtime.NewMockService(ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(errTest).
		Times(1)

	return testSpec{
		ss:  []runtime.ServiceFactory{factoryFor(svc)},
		log: mock_ww.NewMockLogger(ctrl),
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			require.NoError(t, runtime.Start(cfg, lx))
			assert.EqualError(t, lx.Start(context.Background()), errTest.Error(),
				"start error was not reported")
		},
	}
}

func testStopFailure(ctrl *gomock.Controller) testSpec {
	log := mock_ww.NewMockLogger(ctrl)

	log.EXPECT().
		With(matchLoggable).
		Return(log).
		Times(2)

	log.EXPECT().
		Debug("service started").
		Times(1)

	log.EXPECT().
		WithError(errTest).
		Return(log).
		Times(1)

	log.EXPECT().
		Debug("unclean shutdown").
		Times(1)

	svc := mock_runtime.NewMockService(ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(errTest).
		Times(1)

	return testSpec{
		ss:  []runtime.ServiceFactory{factoryFor(svc)},
		log: log,
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			require.NoError(t, runtime.Start(cfg, lx))
			assert.EqualError(t, lx.RequireStart().Stop(context.Background()), errTest.Error(),
				"stop error was not reported")
		},
	}
}

func testUncleanShutdown(ctrl *gomock.Controller) testSpec {
	log := mock_ww.NewMockLogger(ctrl)
	log.EXPECT().
		With(matchLoggable).
		Return(log).
		Times(2)

	log.EXPECT().
		Debug("service started").
		Times(1)

	log.EXPECT().
		WithError(errTest).
		Return(log).
		Times(1)

	log.EXPECT().
		Debug("unclean shutdown").
		Times(1)

	svc := mock_runtime.NewMockService(ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(errTest).
		Times(1)

	return testSpec{
		ss:  []runtime.ServiceFactory{factoryFor(svc)},
		log: log,
		run: func(t *testing.T, lx *fxtest.Lifecycle, cfg runtime.Config) {
			require.NoError(t, runtime.Start(cfg, lx))
			assert.EqualError(t, lx.RequireStart().Stop(context.Background()), errTest.Error(),
				"error from Service.Stop() was not reported")
		},
	}
}

type factoryFunc func() (runtime.Service, error)

func factoryFor(s runtime.Service) factoryFunc {
	return factoryFunc(func() (runtime.Service, error) { return s, nil })
}

func errFactory(err error) factoryFunc {
	return factoryFunc(func() (runtime.Service, error) { return nil, err })
}

func (f factoryFunc) NewService() (runtime.Service, error) { return f() }

type producerServiceFactory struct {
	runtime.ServiceFactory
	runtime.EventProducer
}

func produces(ctrl *gomock.Controller, svc runtime.Service, evts ...interface{}) runtime.ServiceFactory {
	ep := mock_runtime.NewMockEventProducer(ctrl)
	ep.EXPECT().
		Produces().
		Return(evts).
		AnyTimes()

	return producerServiceFactory{
		ServiceFactory: factoryFor(svc),
		EventProducer:  ep,
	}
}

type consumerServiceFactory struct {
	runtime.ServiceFactory
	runtime.EventConsumer
}

func consumes(ctrl *gomock.Controller, svc runtime.Service, evts ...interface{}) runtime.ServiceFactory {
	ec := mock_runtime.NewMockEventConsumer(ctrl)
	ec.EXPECT().
		Consumes().
		Return(evts).
		AnyTimes()

	return consumerServiceFactory{
		ServiceFactory: factoryFor(svc),
		EventConsumer:  ec,
	}
}
