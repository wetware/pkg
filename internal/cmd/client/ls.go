package client

import (
	"errors"
	"fmt"

	"github.com/urfave/cli/v2"
)

func lsFlags() []cli.Flag {
	return []cli.Flag{}
}

func lsInit() cli.BeforeFunc {
	return func(c *cli.Context) error {
		return host.Start()
	}
}

func lsShutdown() cli.AfterFunc {
	return func(c *cli.Context) error {
		return host.Close()
	}
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

		for _, id := range host.Ls() {
			fmt.Printf("/%s\n", id)
		}

		return nil
		// -- DEBUG

		/* TODO:

		path := anchorpath.Split(c.Args().First())

		anchor := host.Walk(path)
		anchor.Ls()

		*/

	}
}
