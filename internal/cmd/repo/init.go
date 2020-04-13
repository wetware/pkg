package repo

import (
	"io/ioutil"

	repoutil "github.com/lthibault/wetware/internal/util/repo"
	"github.com/urfave/cli/v2"
)

func initFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "keysize",
			Aliases: []string{"ks"},
			Usage:   "size of cryptographic keys (must be power of 2)",
			Value:   repoutil.DefaultKeySize,
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Aliases: []string{"q"},
			Usage:   "suppress output on stdout",
		},
	}
}

func initAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		return repoutil.InitRepo(c.Args().First(),
			repoutil.WithKeySize(c.Int("keysize")),
			maybePrint(c))
	}
}

func maybePrint(c *cli.Context) repoutil.Option {
	if c.Bool("quiet") {
		return repoutil.WithPrinter(ioutil.Discard)
	}

	return repoutil.WithPrinter(c.App.Writer)
}
