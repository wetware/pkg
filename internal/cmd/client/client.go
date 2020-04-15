package client

import (
	"github.com/urfave/cli/v2"

	logutil "github.com/lthibault/wetware/internal/util/log"
	ww "github.com/lthibault/wetware/pkg"
)

var host *ww.Host

// Init the wetware client
func Init() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		host, err = ww.New(
			ww.WithLogger(logutil.New(c)),
			ww.WithClientMode())
		return
	}
}

// Commands under `client`
func Commands() []*cli.Command {
	return []*cli.Command{{
		Name:      "ls",
		Usage:     "list cluster elements",
		ArgsUsage: "path",
		Flags:     lsFlags(),
		Before:    lsInit(),
		Action:    lsAction(),
		After:     lsShutdown(),
	}}
}
