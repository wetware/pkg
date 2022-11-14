package debug

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/debug"
)

func profile() *cli.Command {
	return &cli.Command{
		Name:      "profile",
		Usage:     "profile a live node with pprof",
		ArgsUsage: "<peer>",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:  "out",
				Usage: "output file",
			},
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "pprof profile to employ",
				Value:   "cpu",
			},
			&cli.DurationFlag{
				Name:    "duration",
				Aliases: []string{"dur"},
				Usage:   "sampling duration for CPU profile",
				Value:   time.Second,
			},
			&cli.UintFlag{
				Name:  "debug",
				Usage: "debug level for pprof",
			},
			&cli.BoolFlag{
				Name:    "stdout",
				Aliases: []string{"s"},
				Usage:   "output samples to stdout",
			},
		},
		Action: runPprof(),
	}
}

func runPprof() cli.ActionFunc {
	return func(c *cli.Context) error {
		// a, release := node.Walk(c.Context, target(c))
		// defer release()

		// d, release := anchor.Host(a).Debug(c.Context)
		// defer release()

		// TEST
		d, release := node.Debug(c.Context)
		defer release()
		// -- TEST

		name := c.String("profile")
		prof := debug.ProfileFromString(name)

		w, err := writer(c)
		if err != nil {
			return err
		}
		defer w.Close()

		client, release := d.Profiler(c.Context, prof)
		defer release()

		switch prof {
		case debug.InvalidProfile:
			return fmt.Errorf("invalid profile: %s", name)

		case debug.ProfileCPU:
			return debug.
				Sampler(client).
				Sample(c.Context, w, c.Duration("dur"))

		default:
			b, err := debug.
				Snapshotter(client).
				Snapshot(c.Context, uint8(c.Uint("debug")))
			if err == nil {
				_, err = io.Copy(w, bytes.NewReader(b))
			}
			return err
		}
	}
}
