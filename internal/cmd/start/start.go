package start

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/lthibault/wetware/internal/boot"
	ww "github.com/lthibault/wetware/pkg"
)

// Run the `start` command
func Run() cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		var host ww.Host

		app := fx.New(
			fx.NopLogger,       // disable Fx logging
			boot.Provide(c),    // inject dependencies
			fx.Populate(&host), // instantiate `host`
		)

		if err = start(app); err != nil {
			return err
		}
		defer stop(app)

		return run(host)
	}
}

func run(h ww.Host) error {
	return errors.New("NOT IMPLEMENTED")
}

func start(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return app.Start(ctx)
}

func stop(app *fx.App) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return app.Stop(ctx)
}
