package client

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/client"
)

func Ls() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "list anchor elements",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Usage:   "print results as json",
				Value:   false,
				EnvVars: []string{"OUTPUT_JSON"},
			},
		},
		Before: setup(),
		After:  teardown(),
		Action: ls(),
	}
}

func ls() cli.ActionFunc {
	return func(c *cli.Context) error {
		var it = node.Ls(c.Context)

		if c.Bool("json") {
			return lsJSON(c, it)
		}

		lsText(c, it)

		return nil
	}
}

func lsJSON(c *cli.Context, it client.Iterator) error {
	var paths []string

	for it.Next() {
		paths = append(paths, it.Anchor().Path())
	}

	return json.NewEncoder(c.App.Writer).Encode(paths)
}

func lsText(c *cli.Context, it client.Iterator) {
	for it.Next() {
		fmt.Println(it.Anchor().Path())
	}
}
