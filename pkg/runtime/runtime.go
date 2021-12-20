package runtime

import (
	"context"
	"fmt"
	"io"

	"github.com/lthibault/log"
	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"
)

/****************************************************************************
 *                                                                          *
 *  runtime.go is responsible for managing the lifetimes of services.       *
 *                                                                          *
 ****************************************************************************/

// Bind a context to an Fx.Option that loads dependencies at runtime.
func Bind() fx.Option {
	return fx.Options(
		network,
		system,
		fx.Invoke(bind))
}

// Config declares dependencies that are dynamically resolved at
// runtime.
type Config struct {
	fx.In

	Lifecycle fx.Lifecycle

	Logger log.Logger

	Supervisor *suture.Supervisor
	Services   []suture.Service `group:"services"` // caller-supplied services
}

func bind(c *cli.Context, config Config) {
	ctx, cancel := context.WithCancel(c.Context) // cancelled by stop hook

	// Bind user-defined services to the runtime.
	for _, service := range config.Services {
		config.Supervisor.Add(service)
	}

	// Set up some shared variables.  These will mutate throughout the application
	// lifecycle.
	var cherr <-chan error

	// Hook main service (Supervisor) into application lifecycle.
	config.Lifecycle.Append(fx.Hook{
		// Bind global variables and start wetware.
		OnStart: func(_ context.Context) error {
			cherr = config.Supervisor.ServeBackground(ctx) // NOTE: application context

			// The wetware environment is now loaded.  Bear in mind
			// that this is a PA/EL system, so we hand over control
			// to the user without synchronizing the cluster view.
			//
			// System events are likewise asynchronous, but reliable.
			// Users can wait for the local node to have successfully
			// bound to network interfaces by subscribing to the
			// 'EvtLocalAddrsUpdated' event.

			return nil
		},
		OnStop: func(ctx context.Context) (err error) {
			// Cancel the application context.
			//
			// Beyond this point, the services in this local
			// process are no longer guaranteed to be in sync.
			cancel()
			config.Logger.Tracef("runtime context expired")

			// Wait for the supervisor to shut down gracefully.
			// If it does not terminate in a timely fashion,
			// abort the application and return an error.
			select {
			case err = <-cherr:
				return err

			case <-ctx.Done():
				return fmt.Errorf("shutdown: %w", ctx.Err())
			}
		},
	})
}

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
