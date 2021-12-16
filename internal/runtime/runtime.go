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
		fx.Provide(
			bindSystem,
			bindNetwork),
		fx.Invoke(bind))
}

// Config declares dependencies that are dynamically resolved at
// runtime.
type Config struct {
	fx.In

	Lifecycle fx.Lifecycle

	CLI    *cli.Context
	Logger log.Logger

	Supervisor *suture.Supervisor
	Services   []suture.Service `group:"services"` // caller-supplied services
}

// Initialize a context from the configuration.  This method MUST be
// idempotent.
func (config *Config) Init() context.Context {

	//
	// Adding and removing capabilities can be achieved
	// by 'Bind'ing Wetware modules to an Fx runtime.
	//
	// Fx's dependency injection runtime provides the basis for integrating
	// capability-based security into Wetware.  Each module binds a copy of
	// it's main config struct to  the context.  This ensures that packages
	// cannot access it without first importing the 'runtime' package. This
	// property facilitates static analysis because the Go dependency graph
	// is equal to the object authority graph.
	//
	return context.WithValue(config.CLI.Context, (*Config)(nil), config)
}

func bind(config Config) {
	ctx, cancel := context.WithCancel(config.Init())
	defer cancel()

	// Bind user-defined services to the runtime.
	for _, service := range config.Services {
		config.Supervisor.Add(service)
	}

	// Set up some shared variables.  These will mutate throughout the application
	// lifecycle.
	var signal = make(sigchan, 1)

	// Hook main service (Supervisor) into application lifecycle.
	config.Lifecycle.Append(fx.Hook{
		// Bind global variables and start wetware.
		OnStart: func(_ context.Context) error {
			go signal.Error(config.Supervisor.Serve(ctx)) // NOTE: application context

			// The wetware environment is now loaded.  Bear in mind
			// that this is a PA/EL system, so we hand over control
			// to the user without synchronizing the cluster view.
			//
			// System events are likewise asynchronous, but reliable.
			// Users can wait for the local node to have successfully
			// bound to network interfaces by subscribing to the
			// 'EvtLocalAddrsUpdated' event.
			config.Logger.Info("wetware loaded")

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
			case err = <-signal:
				return err

			case <-ctx.Done():
				return fmt.Errorf("shutdown: %w", ctx.Err())
			}
		},
	})
}

// a channel that can signal exceptions
type sigchan chan error

func (ch sigchan) Error(err error) { ch <- err }
func (ch sigchan) Success()        { close(ch) }

func closer(c io.Closer) fx.Hook {
	return fx.Hook{
		OnStop: func(context.Context) error {
			return c.Close()
		},
	}
}
