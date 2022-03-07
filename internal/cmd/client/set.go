package client

import (
	"errors"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/cap/cluster"
	"github.com/wetware/ww/pkg/client"
	"github.com/wetware/ww/pkg/vat"
)

func Set() *cli.Command {
	var (
		v vat.Network
		n *client.Node
	)

	return &cli.Command{
		Name:   "set",
		Usage:  "set data in cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(&v, &n),
		Action: set(&n),
		After:  afterAnchor(&v),
	}
}

func set(nn **client.Node) cli.ActionFunc {
	return func(c *cli.Context) error {
		n := *nn
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		a, err := n.Walk(c.Context, path)
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
