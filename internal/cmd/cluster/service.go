package cluster

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/service"
)

func discover() *cli.Command {
	return &cli.Command{
		Name:    "discover",
		Aliases: []string{"disc"},
		Usage:   "discover a service",
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
		Action: discAction(),
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
				Usage:   "multiaddress of the service provdier",
			},
		},
		Before: setup(),
		After:  teardown(),
		Action: provAction(),
	}
}

func discAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		disc, release := node.Service(c.Context)
		defer release()

		locator, release := disc.Locator(c.Context, c.String("name"))
		defer release()

		locs, release := locator.FindProviders(c.Context)
		defer release()

		for loc, ok := locs.Next(); ok; loc, ok = locs.Next() {
			fmt.Println(loc.String())
		}

		return locs.Err()
	}
}

func provAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		disc, release := node.Service(c.Context)
		defer release()

		serviceId := c.String("name")

		provider, release := disc.Provider(c.Context, serviceId)
		defer release()

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

		fut, release := provider.Provide(c.Context, loc)
		defer release()

		fmt.Printf("providing |%s| at", serviceId)
		for _, maddr := range maddrs {
			fmt.Printf(" %s", maddr.String())
		}
		fmt.Println()

		return fut.Await(c.Context)
	}
}
