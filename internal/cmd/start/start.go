package start

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	logutil "github.com/lthibault/wetware/internal/util/log"
	ww "github.com/lthibault/wetware/pkg"
)

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
	return func(c *cli.Context) (err error) {
		var h *ww.Host
		if h, err = ww.New(ww.WithLogger(logutil.New(c))); err != nil {
			return err
		}

		if err = h.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}

		return wait(h)
	}
}

func wait(h *ww.Host) error {
	h.Log().Info("host started")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	h.Log().Warn("host shutting down")
	return h.Close()
}
