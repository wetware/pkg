package start

import (
	"context"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	ctxutil "github.com/lthibault/wetware/internal/util/ctx"
	logutil "github.com/lthibault/wetware/internal/util/log"
	"github.com/lthibault/wetware/pkg/server"
)

var (
	ctx  = ctxutil.WithDefaultSignals(context.Background())
	host server.Host
)

// Init the `start` command
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		host, err = server.New(
			server.WithLogger(logutil.New(c)),
		)

		return
	}
}

// Flags for the `start` command
func Flags() []cli.Flag {
	return []cli.Flag{}
}

// Run the `start` command
func Run() cli.ActionFunc {
	return func(c *cli.Context) error {
		if err := host.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}

		host.Log().Info("host started")
		<-ctx.Done()
		host.Log().Warn("host shutting down")

		if err := host.Close(); err != nil {
			return errors.Wrap(err, "stop host")
		}

		return nil
	}
}
