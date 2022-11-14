package debug

import (
	"time"

	"github.com/urfave/cli/v2"
)

func trace() *cli.Command {
	return &cli.Command{
		Name:      "trace",
		Usage:     "perform a runtime trace on a live host",
		ArgsUsage: "<peer>",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:  "out",
				Usage: "output file",
			},
			&cli.DurationFlag{
				Name:    "duration",
				Aliases: []string{"dur"},
				Usage:   "sampling duration for CPU profile",
				Value:   time.Second,
			},
			&cli.BoolFlag{
				Name:    "stdout",
				Aliases: []string{"s"},
				Usage:   "output samples to stdout",
			},
		},
		Action: runTrace(),
	}
}

func runTrace() cli.ActionFunc {
	return func(c *cli.Context) error {
		// a, release := node.Walk(c.Context, target(c))
		// defer release()

		// d, release := anchor.Host(a).Debug(c.Context)
		// defer release()

		// TEST
		d, release := node.Debug(c.Context)
		defer release()
		// -- TEST

		tracer, release := d.Tracer(c.Context)
		defer release()

		w, err := writer(c)
		if err != nil {
			return err
		}
		defer w.Close()

		return tracer.Sample(c.Context, w, c.Duration("dur"))
	}
}
