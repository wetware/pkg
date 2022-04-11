package client

import (
	"fmt"
	"path"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/client"
)

func Ls() *cli.Command {
	return &cli.Command{
		Name:   "ls",
		Usage:  "list anchor elements",
		Action: ls(),
	}
}

func ls() cli.ActionFunc {
	return func(c *cli.Context) error {
		it := node.Ls(c.Context)
		for it.Next() {
			fmt.Println(pathString(it.Anchor()))
		}

		return it.Err()
	}
}

func pathString(a client.Anchor) string {
	return path.Clean(fmt.Sprintf("/%s", strings.Join(a.Path(), "/")))
}
