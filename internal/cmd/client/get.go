package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/cap/cluster"
)

func Get() *cli.Command {
	return &cli.Command{
		Name:   "get",
		Usage:  "set data in cluster path",
		Flags:  clientFlags,
		Before: beforeAnchor(),
		Action: get(),
		After:  afterAnchor(),
	}
}

func get() cli.ActionFunc {
	return func(c *cli.Context) error {
		path := cleanPath(strings.Split(c.Args().First(), "/"))
		a, err := node.Walk(c.Context, path)
		if err != nil {
			return err
		}
		defer a.Release(c.Context)

		co, ok := a.(cluster.Container)
		if !ok {
			return errors.New("path is not gettable")
		}

		data, err := co.Get(c.Context)
		if err != nil {
			return err
		}

		fmt.Println(string(data))

		return nil
	}
}
