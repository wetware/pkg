package runtime_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

/*
	Test Functions
*/

func TestLifecycleSuccessSuite(t *testing.T) { suite.Run(t, new(LifecycleSuccess)) }
func TestLifecycleFailureSuite(t *testing.T) { suite.Run(t, new(LifecycleFailure)) }
func TestFactoryFailureSuite(t *testing.T)   { suite.Run(t, new(FactoryFailure)) }

/*
	Test Suites
*/

type LifecycleSuccess struct{ runtimeTestSuite }

func (suite *LifecycleSuccess) AfterTest(_, _ string) {
	defer suite.ctrl.Finish()

	ctx, cancel := suite.Context()
	defer cancel()

	suite.Require().NoError(suite.Exec())
	suite.NoError(suite.lx.Start(ctx), "lifecycle start failed")
	suite.NoError(suite.lx.Stop(ctx), "lifecycle stop failed")
}

func (suite *LifecycleSuccess) TestNoServices() { /* no services */ }

func (suite *LifecycleSuccess) TestNoDependencies() {
	suite.log.EXPECT().
		With(matchLoggable).
		Return(suite.log).
		Times(1)

	suite.log.EXPECT().
		Debug("service started").
		Times(1)

	svc := mock_runtime.NewMockService(suite.ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(nil).
		Times(1)

	suite.AddFactories(factoryFor(svc))
}

func (suite *LifecycleSuccess) TestDependencies() {
	suite.log.EXPECT().
		With(matchLoggable).
		Return(suite.log).
		Times(2)

	suite.log.EXPECT().
		Debug("service started").
		Times(2)

	svc := mock_runtime.NewMockService(suite.ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(2)

	svc.EXPECT().
		Stop(matchContext).
		Return(nil).
		Times(2)

	suite.AddFactories(
		produces(suite.ctrl, svc, struct{}{}),
		consumes(suite.ctrl, svc, struct{}{}),
	)
}

type FactoryFailure struct {
	runtimeTestSuite
	expect error
}

func (suite *FactoryFailure) TearDownTest() { suite.expect = nil }

func (suite *FactoryFailure) AfterTest(_, _ string) {
	defer suite.ctrl.Finish()

	// check that the test is properly defined.
	if suite.expect == nil {
		suite.T().Error("TEST NOT IMPLEMENTED (expected error not specified)")
	}

	assert.EqualError(suite.T(),
		suite.Exec(),
		suite.expect.Error(),
		"unexpected factory error")
}

func (suite *FactoryFailure) TestFactoryError() {
	suite.expect = errTest
	suite.AddFactories(
		errFactory(errTest),
		factoryFor(mock_runtime.NewMockService(suite.ctrl)),
	)
}

func (suite *FactoryFailure) TestUnresolvedDependency() {
	suite.expect = runtime.DependencyError{reflect.TypeOf(struct{}{})}
	suite.AddFactories(
		consumes(suite.ctrl, mock_runtime.NewMockService(suite.ctrl), struct{}{}),
	)
}

type LifecycleFailure struct {
	runtimeTestSuite
	startErr, stopErr error
}

func (suite *LifecycleFailure) TearDownTest() {
	suite.startErr = nil
	suite.stopErr = nil
}

func (suite *LifecycleFailure) AfterTest(_, _ string) {
	defer suite.ctrl.Finish()

	ctx, cancel := suite.Context()
	defer cancel()

	// check that the test is properly defined.
	if suite.startErr == nil && suite.stopErr == nil {
		suite.T().Error("TEST NOT IMPLEMENTED (must define startErr and/or stopErr)")
	}

	suite.Require().NoError(suite.Exec())

	if err := suite.lx.Start(ctx); suite.startErr != nil {
		suite.EqualError(err, suite.startErr.Error(),
			"unexpected lifecycle start error")
	} else {
		suite.NoError(err, "unexpected lifecycle start error")
	}

	if suite.stopErr != nil {
		suite.EqualError(suite.lx.Stop(ctx), suite.stopErr.Error(),
			"unexpected lifecycle stop error")
	}
}

func (suite *LifecycleFailure) TestStartError() {
	suite.startErr = errTest

	svc := mock_runtime.NewMockService(suite.ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(errTest).
		Times(1)

	suite.AddFactories(factoryFor(svc))
}

func (suite *LifecycleFailure) TestStopError() {
	suite.stopErr = errTest

	suite.log.EXPECT().
		With(matchLoggable).
		Return(suite.log).
		Times(2)

	suite.log.EXPECT().
		Debug("service started").
		Times(1)

	suite.log.EXPECT().
		WithError(errTest).
		Return(suite.log).
		Times(1)

	suite.log.EXPECT().
		Debug("unclean shutdown").
		Times(1)

	svc := mock_runtime.NewMockService(suite.ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(errTest).
		Times(1)

	suite.AddFactories(factoryFor(svc))
}

func (suite *LifecycleFailure) TestUncleanShutdown() {
	suite.stopErr = errTest

	suite.log.EXPECT().
		With(matchLoggable).
		Return(suite.log).
		Times(2)

	suite.log.EXPECT().
		Debug("service started").
		Times(1)

	suite.log.EXPECT().
		WithError(errTest).
		Return(suite.log).
		Times(1)

	suite.log.EXPECT().
		Debug("unclean shutdown").
		Times(1)

	svc := mock_runtime.NewMockService(suite.ctrl)
	svc.EXPECT().
		Start(matchContext).
		Return(nil).
		Times(1)

	svc.EXPECT().
		Stop(matchContext).
		Return(errTest).
		Times(1)

	suite.AddFactories(factoryFor(svc))
}

/*
	Utilities
*/

type runtimeTestSuite struct {
	suite.Suite

	ctrl *gomock.Controller
	log  *mock_ww.MockLogger

	lx  *fxtest.Lifecycle
	cfg runtime.Config
}

func (suite *runtimeTestSuite) Context() (context.Context, context.CancelFunc) {
	t, ok := suite.T().Deadline()
	if !ok {
		return context.WithCancel(context.Background())
	}

	return context.WithDeadline(context.Background(), t)
}

func (suite *runtimeTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.log = mock_ww.NewMockLogger(suite.ctrl)

	suite.lx = fxtest.NewLifecycle(suite.T())
	suite.cfg = runtime.Config{Log: suite.log}
}

func (suite *runtimeTestSuite) Exec() error { return runtime.Start(suite.cfg, suite.lx) }

func (suite *runtimeTestSuite) AddFactories(fs ...runtime.ServiceFactory) {
	suite.cfg.Services = append(suite.cfg.Services, fs...)
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
