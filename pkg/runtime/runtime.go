package runtime

import (
	"context"

	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/event"
)

// Bundle services to define application-specific configurations
// (see pkg/client/runtime.go for a simple example).
func Bundle(ps ...ServiceProvider) ServiceBundle {
	return ps
}

// Register runtime
func Register(bus event.Bus, b ServiceBundle, lx fx.Lifecycle) error {
	return new(runtime).
		bindServices(b).
		bindEventBus(bus).
		bindLifecycle(lx).
		Error
}

// ServiceBundle is a set of ServiceFactories that are started/stopped in a
// well-orchestrated manner.
type ServiceBundle []ServiceProvider

// ServiceProvider is a constructor for a service.
type ServiceProvider interface {
	Service() (Service, error)
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

// ErrorStreamer reports errors asynchronously
type ErrorStreamer interface {
	// Errors encountered during the course of execution of the service.
	// By convention, runtime services must automatically recover from internal errors.
	// Unrecoverable errors are reported through panics.
	Errors() <-chan error

	Loggable() map[string]interface{}
}

type runtime struct {
	Error error
	ss    []Service
	hs    []fx.Hook
}

func (r *runtime) bindServices(b ServiceBundle) *runtime {
	if r.Error == nil {
		r.ss = make([]Service, len(b))
		for i, p := range b {
			if r.ss[i], r.Error = p.Service(); r.Error != nil {
				break
			}
		}
	}

	return r
}

func (r *runtime) bindEventBus(bus event.Bus) *runtime {
	if r.Error == nil {
		for _, svc := range r.ss {
			r.hs = append(r.hs, hook(bus, svc))
		}
	}

	return r
}

func (r *runtime) bindLifecycle(lx fx.Lifecycle) *runtime {
	if r.Error == nil {
		for _, h := range r.hs {
			lx.Append(h)
		}
	}

	return r
}

func hook(bus event.Bus, s Service) fx.Hook {
	return fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := s.Start(ctx); err != nil {
				return err
			}

			if err := streamErrs(bus, s); err != nil {
				return err
			}

			return emit(bus, new(EvtServiceStateChanged),
				EvtServiceStateChanged{loggable: s, State: ServiceStateStarting})
		},
		OnStop: func(ctx context.Context) error {
			if err := s.Stop(ctx); err != nil {
				return err
			}

			return emit(bus, new(EvtServiceStateChanged),
				EvtServiceStateChanged{loggable: s, State: ServiceStateStopping})
		},
	}
}

func streamErrs(bus event.Bus, s Service) error {
	if es, ok := s.(ErrorStreamer); ok {
		return notify(bus, es)
	}

	return nil
}

func emit(bus event.Bus, tpe, v interface{}) error {
	e, err := bus.Emitter(tpe)
	if err != nil {
		return err
	}
	defer e.Close()

	return e.Emit(v)
}

func notify(bus event.Bus, es ErrorStreamer) error {
	e, err := bus.Emitter(new(Exception))
	if err == nil {
		go watch(e, es)
	}

	return err
}

func watch(e event.Emitter, es ErrorStreamer) {
	for err := range es.Errors() {
		if err = e.Emit(Exception{
			error: err,
			fs:    es.Loggable(),
		}); err != nil {
			panic(err)
		}
	}
}
