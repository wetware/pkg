package client

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

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
		Action: ls(),
	}
}

func ls() cli.ActionFunc {
	return func(c *cli.Context) error {
		paths := make([]string, 0)
		it := node.Ls(c.Context)
		for it.Next() {
			paths = append(paths, pathString(it.Anchor()))
		}

		if c.Bool("json") {
			jsonOutput, err := json.Marshal(paths)
			if err != nil {
				return nil
			}
			fmt.Println(string(jsonOutput))
		} else {
			for _, path := range paths {
				fmt.Println(path)
			}
		}

		return it.Err()
	}
}

func pathString(a client.Anchor) string {

	return path.Clean(fmt.Sprintf("/%s", strings.Join(a.Path(), "/")))
}
