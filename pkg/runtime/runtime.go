package runtime

import (
	"context"

	ww "github.com/wetware/ww/pkg"
	"go.uber.org/fx"
)

// Config specifies a set of runtime services.
type Config struct {
	fx.In

	Log      ww.Logger
	Services []ServiceFactory `group:"runtime"`
}

// Start a runtime in the background
func Start(cfg Config, lx fx.Lifecycle) (err error) {
	var svc Service
	for _, factory := range cfg.Services {
		if svc, err = factory.NewService(); err != nil {
			break
		}

		lx.Append(hook(cfg.Log, svc))
	}

	return
}

// ServiceFactory is a constructor for a service.
type ServiceFactory interface {
	NewService() (Service, error)
	// Consumes() []interface{}
	// Emits() []interface{}
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
