package runtime

import (
	"context"

	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

// Bundle services to define application-specific configurations
// (see pkg/client/runtime.go for a simple example).
func Bundle(ps ...ServiceProvider) ServiceBundle {
	return ps
}

// Register runtime
func Register(log ww.Logger, b ServiceBundle, lx fx.Lifecycle) (err error) {
	var svc Service
	for _, p := range b {
		if svc, err = p.Service(); err != nil {
			break
		}

		lx.Append(hook(log, svc))
	}

	return
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
