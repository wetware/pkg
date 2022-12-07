package cluster

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/wetware/ww/pkg/discovery"
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
		disc, release := node.Discovery(c.Context)
		defer release()

		locator, release := disc.Locator(c.Context, c.String("name"))
		defer release()

		addrs, release := locator.FindProviders(c.Context)
		defer release()

		for addr, ok := addrs.Next(); ok; addr, ok = addrs.Next() {
			fmt.Println(addr)
		}

		return addrs.Err()
	}
}

func provAction() cli.ActionFunc {
	return func(c *cli.Context) error {
		disc, release := node.Discovery(c.Context)
		defer release()

		provider, release := disc.Provider(c.Context, c.String("name"))
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

		addr := discovery.Addr{Maddrs: maddrs}
		fut, release := provider.Provide(c.Context, addr)
		defer release()

		fmt.Println("providing...")

		return fut.Await(c.Context)
	}
}
