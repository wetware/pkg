//go:generate mockgen -destination ../../internal/test/mock/pkg/runtime/mock_runtime.go github.com/wetware/ww/pkg/runtime ServiceFactory,EventProducer,EventConsumer,Service

package runtime

import (
	"context"
	"fmt"
	"reflect"

	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

// DependencyError is returned if an event consumed by a registered Service does not
// have a corresponding producer registered to the runtime.
type DependencyError struct {
	Type reflect.Type
}

func (err DependencyError) Error() string {
	return fmt.Sprintf("no producer registered for event '%s'", err.Type)
}

// ServiceFactory is a constructor for a service.
type ServiceFactory interface {
	NewService() (Service, error)
}

// EventProducer is an optional interface implemented by ServiceFactory that declares
// which events a given service produces.  It is used to verify event dependencies.
type EventProducer interface {
	Produces() []interface{}
}

// EventConsumer is an optional interface implemented by ServiceFactory that declares
// which events a given service consumes.  It is used to verify event dependencies.
type EventConsumer interface {
	Consumes() []interface{}
}

// Service is a process that runs in the background.
// A set of services constitutes a "runtime environment".  Different wetware objects,
// such as Clients and Hosts, have their own runtimes.
type Service interface {
	// Start the service.  The startup sequence is aborted if the context expires.
	Start(context.Context) error

	// Stop the service.  The shutdown sequence is aborted, resulting in an unclean
	// shutdown, if the context expires.
	Stop(context.Context) error

	// Loggable representation of the service
	Loggable() map[string]interface{}
}

// Config specifies a set of runtime services.
type Config struct {
	fx.In

	Log      ww.Logger
	Services []ServiceFactory `group:"runtime"`
}

// Start a runtime in the background
func Start(cfg Config, lx fx.Lifecycle) (err error) {
	var loader serviceLoader
	for _, factory := range cfg.Services {
		loader.LoadService(lx, cfg.Log, factory)
	}

	return loader.Error()
}

type serviceLoader struct {
	err        error
	prod, cons map[reflect.Type]struct{}
}

func (sl *serviceLoader) Error() error {
	if sl.err != nil {
		return sl.err
	}

	for ev := range sl.cons {
		if _, ok := sl.prod[ev]; !ok {
			return DependencyError{ev}
		}
	}

	return nil
}

func (sl *serviceLoader) LoadService(lx fx.Lifecycle, log ww.Logger, factory ServiceFactory) {
	if sl.err != nil {
		return
	}

	var svc Service
	if svc, sl.err = factory.NewService(); sl.err == nil {
		lx.Append(hook(log, svc))
		sl.addDependencies(factory)
	}
}

func (sl *serviceLoader) addDependencies(f ServiceFactory) {
	if ep, ok := f.(EventProducer); ok {
		sl.addEventProducer(ep)
	}

	if ec, ok := f.(EventConsumer); ok {
		sl.addEventConsumer(ec)
	}
}

func (sl *serviceLoader) addEventProducer(ep EventProducer) {
	if sl.prod == nil {
		sl.prod = map[reflect.Type]struct{}{}
	}

	for _, ev := range ep.Produces() {
		sl.prod[reflect.TypeOf(ev)] = struct{}{}
	}
}

func (sl *serviceLoader) addEventConsumer(ec EventConsumer) {
	if sl.cons == nil {
		sl.cons = map[reflect.Type]struct{}{}
	}

	for _, ev := range ec.Consumes() {
		sl.cons[reflect.TypeOf(ev)] = struct{}{}
	}
}

func hook(log ww.Logger, svc Service) fx.Hook {
	return fx.Hook{
		OnStart: func(ctx context.Context) (err error) {
			if err = svc.Start(ctx); err == nil {
				log.With(svc).Debug("service started")
			}

			return
		},
		OnStop: func(ctx context.Context) (err error) {
			if err = svc.Stop(ctx); err != nil {
				log.With(svc).WithError(err).Debug("unclean shutdown")
			}

			return
		},
	}
}
