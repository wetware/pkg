package client

import (
	"fmt"
	"io"

	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func lsFlags() []cli.Flag {
	return []cli.Flag{}
}

func lsAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		if err := validatePath(c.Args().First()); err != nil {
			return errors.Wrap(err, "invalid path")
		}

		// TODO:  avoid extra round-trip.
		anchor, err := root.Walk(proc, anchorpath.Parts(c.Args().First()))
		if err != nil {
			return errors.Wrapf(err, "walk %s", c.Args().First())
		}

		return printPaths(c.App.Writer, anchor.Ls(proc))
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

func printPaths(w io.Writer, it ww.Iterator) error {
	for it.Next() {
		fmt.Fprintf(w, "/%s\n", it.Path())
	}

	return it.Err()
}
