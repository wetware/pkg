package start

import (
	log "github.com/lthibault/log/pkg"
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
		&cli.StringFlag{
			Name:    "logfmt",
			Aliases: []string{"f"},
			Usage:   "text, json, none",
			Value:   "text",
			EnvVars: []string{"CASM_LOGFMT"},
		},
		&cli.StringFlag{
			Name:    "loglvl",
			Usage:   "trace, debug, info, warn, error, fatal",
			Value:   "info",
			EnvVars: []string{"CASM_LOGLVL"},
		},

		/************************
		*	undocumented flags	*
		*************************/
		&cli.BoolFlag{
			Name:    "prettyprint",
			Aliases: []string{"pp"},
			Usage:   "pretty-print JSON output",
			Hidden:  true,
		},
	}
}

// Run the `start` command
func Run() cli.ActionFunc {
	return func(c *cli.Context) (err error) {
		log := logutil.New(c)

		var h *ww.Host
		if h, err = ww.New(ww.WithLogger(log)); err != nil {
			return err
		}

		if err = h.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}
		defer stop(log, h)

		return errors.New("NOT IMPLEMENTED")
	}
}

func stop(log log.Logger, h *ww.Host) {
	if err := h.Close(); err != nil {
		log.WithError(err).Fatal("error encountered during host shutdown")
	}
}
