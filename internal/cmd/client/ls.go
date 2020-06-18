package client

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

func lsFlags() []cli.Flag {
	return []cli.Flag{}
}

func lsAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		path := c.Args().First()

		if err := validatePath(path); err != nil {
			return errors.Wrap(err, "invalid path")
		}

		cs, err := root.Walk(ctx, anchorpath.Parts(path)).Ls(ctx)
		if err != nil {
			return errors.Wrapf(err, "ls %s", path)
		}

		for _, anchor := range cs {
			fmt.Fprintf(c.App.Writer, "/%s\n", anchor)
		}

		return nil
	}
}

func validatePath(path string) error {
	if path == "" {
		return errors.New("must be a glob argument")
	}

	if path[0] != '/' {
		return errors.New("must specify absolute path")
	}

	return nil
}
