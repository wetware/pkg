package client

import (
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

func Walk() *cli.Command {
	var (
		v vat.Network
		n *client.Node
	)

	return &cli.Command{
		Name:   "walk",
		Usage:  "create cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(&v, &n),
		Action: walk(&n),
		After:  afterAnchor(&v),
	}
}

func walk(nn **client.Node) cli.ActionFunc {
	return func(c *cli.Context) error {
		n := *nn
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		anchor, err := n.Walk(c.Context, path)
		if err != nil {
			return err
		}

		defer anchor.Release(c.Context)

		return err
	}
}
