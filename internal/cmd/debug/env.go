package debug

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

func env() *cli.Command {
	return &cli.Command{
		Name:      "env",
		Usage:     "display host environment variables",
		ArgsUsage: "<peer>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Usage:   "print results as json",
				EnvVars: []string{"WW_FMT_JSON"},
			},
		},
		Action: queryEnvVars(),
	}
}

func queryEnvVars() cli.ActionFunc {
	return func(c *cli.Context) error {
		// a, release := node.Walk(c.Context, target(c))
		// defer release()

		// d, release := anchor.Host(a).Debug(c.Context)
		// defer release()

		// TEST
		d, release := node.Debug(c.Context)
		defer release()
		// -- TEST

		env, err := d.EnvVars(c.Context)
		if err != nil {
			return err
		}

		return renderEnvVars(c, env)
	}
}

func renderEnvVars(c *cli.Context, env []string) error {
	if c.Bool("json") {
		return json.NewEncoder(c.App.Writer).Encode(env)
	}

	for _, envvar := range env {
		fmt.Println(envvar)
	}

	return nil
}
