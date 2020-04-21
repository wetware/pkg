package client

import (
	"context"
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
		path, err := parsePath(c.Args().First())
		if err != nil {
			return errors.Wrap(err, "parse path")
		}

		it := cluster.Walk(context.Background(), path).Ls()
		for it.Next() {
			fmt.Printf("/%s\n", it.Path())
		}

		return it.Err()
	}
}

func parsePath(path string) ([]string, error) {
	if path == "" {
		return nil, errors.New("path must be a glob argument")
	}

	if !anchorpath.Abs(path) {
		return nil, errors.New("must specify absolute path")
	}

	return anchorpath.Parts(path), nil
}
