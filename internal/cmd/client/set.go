package client

import (
	"errors"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/cap/cluster"
)

func Set() *cli.Command {
	return &cli.Command{
		Name:   "set",
		Usage:  "set data in cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(),
		Action: set(),
		After:  afterAnchor(),
	}
}

func set() cli.ActionFunc {
	return func(c *cli.Context) error {
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		a, err := node.Walk(c.Context, path)
		if err != nil {
			return err
		}
		defer a.Release(c.Context)

		co, ok := a.(cluster.Container)
		if !ok {
			return errors.New("path is not settable")
		}

		return co.Set(c.Context, []byte(c.Args().Get(1)))
	}
}
