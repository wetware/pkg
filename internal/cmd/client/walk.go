package client

import (
	"strings"

	"github.com/urfave/cli/v2"
)

func Walk() *cli.Command {
	return &cli.Command{
		Name:   "walk",
		Usage:  "create cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(),
		Action: walk(),
		After:  afterAnchor(),
	}
}

func walk() cli.ActionFunc {
	return func(c *cli.Context) error {
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		anchor, err := node.Walk(c.Context, path)
		if err != nil {
			return err
		}

		defer anchor.Release(c.Context)

		return err
	}
}
