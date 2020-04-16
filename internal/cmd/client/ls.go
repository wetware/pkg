package client

import (
	"errors"

	"github.com/urfave/cli/v2"
)

func lsFlags() []cli.Flag {
	return []cli.Flag{}
}

func lsAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		if c.Args().First() == "" {
			return errors.New("path must be a glob argument")
		}

		// DEBUG
		if c.Args().First() != "/" {
			return errors.New("TODO:  implement Anchor.Walk")
		}

		// for _, id := range host.Ls() {
		// 	fmt.Printf("/%s\n", id)
		// }

		// return nil
		return errors.New("NOT IMPLEMENTED")

		// -- DEBUG

		/* TODO:

		path := anchorpath.Split(c.Args().First())

		anchor := host.Walk(path)
		anchor.Ls()

		*/

	}
}
