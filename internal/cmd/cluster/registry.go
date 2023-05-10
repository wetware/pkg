package cluster

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	service "github.com/wetware/ww/pkg/registry"
)

func discover() *cli.Command {
	return &cli.Command{
		Name:    "locate",
		Aliases: []string{"loc"},
		Usage:   "locate a service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "service name",
				Required: true,
			},
		},
		Before: setup(),
		After:  teardown(),
		Action: locAction(),
	}
}

func provide() *cli.Command {
	return &cli.Command{
		Name:    "provide",
		Aliases: []string{"prov"},
		Usage:   "provide a service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "service name",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:    "multiaddr",
				Aliases: []string{"maddr"},
				Usage:   "multiaddress of the service provider",
			},
		},
		Before: setup(),
		After:  teardown(),
		Action: provAction(),
	}
}

func locAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		registry, release := node.Registry(c.Context)
		defer release()

		topic, release := node.Join(c.Context, c.String("name"))
		defer release()

		locs, release := registry.FindProviders(c.Context, topic)
		defer release()

		for loc, ok := locs.Next(); ok; loc, ok = locs.Next() {
			fmt.Println(loc.String())
		}

		return locs.Err()
	}
}

func provAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		// parse multiaddr location
		maddrsStr := c.StringSlice("maddr")
		maddrs := make([]ma.Multiaddr, 0, len(maddrsStr))
		for _, maddrStr := range maddrsStr {
			maddr, err := ma.NewMultiaddr(maddrStr)
			if err != nil {
				return err
			}
			maddrs = append(maddrs, maddr)
		}

		loc, err := service.NewLocation()
		if err != nil {
			return fmt.Errorf("failed to create location: %w", err)
		}

		if err := loc.SetMaddrs(maddrs); err != nil {
			return fmt.Errorf("failed to set maddrs: %w", err)
		}

		if err := loc.SetService(c.String("name")); err != nil {
			return fmt.Errorf("failed to set service name: %w", err)
		}

		// provide service
		registry, release := node.Registry(c.Context)
		defer release()

		topic, release := node.Join(c.Context, c.String("name"))
		defer release()

		fut, release := registry.Provide(c.Context, topic, loc)
		defer release()

		fmt.Printf("providing |%s| at", c.String("name"))
		for _, maddr := range maddrs {
			fmt.Printf(" %s", maddr.String())
		}
		fmt.Println()

		return fut.Await(c.Context)
	}
}
