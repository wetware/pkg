package start

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	logutil "github.com/lthibault/wetware/internal/util/log"
	"github.com/lthibault/wetware/pkg/server"
)

var (
	peer  server.Host
	close <-chan os.Signal
)

// Init the `start` command
func Init() cli.BeforeFunc {
	return func(c *cli.Context) error {
		peer = server.New(server.WithLogger(logutil.New(c)))

		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		close = ch

		return nil
	}
}

// Flags for the `start` command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "repo",
			Aliases: []string{"r"},
			Usage:   "path to IPFS repository",
			EnvVars: []string{"WW_REPO"},
		},
	}
}

// Run the `start` command
func Run() cli.ActionFunc {
	return func(c *cli.Context) error {
		if err := peer.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}

		peer.Log().Info("host started")
		<-close
		peer.Log().Warn("host shutting down")

		if err := peer.Close(); err != nil {
			return errors.Wrap(err, "stop host")
		}

		return nil
	}
}
