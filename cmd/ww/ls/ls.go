package ls

import (
	"fmt"

	"log/slog"

	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/system"
)

func Command() *cli.Command {
	return &cli.Command{
		Name: "ls",
		Action: func(c *cli.Context) error {
			h, err := client.NewHost()
			if err != nil {
				return err
			}
			defer h.Close()

			host, err := system.Bootstrap[host.Host](c.Context, h, client.Dialer{
				Logger:   slog.Default(),
				NS:       c.String("ns"),
				Peers:    c.StringSlice("peer"),
				Discover: c.String("discover"),
			})
			if err != nil {
				return err
			}
			defer host.Release()

			view, release := host.View(c.Context)
			defer release()

			it, release := view.Iter(c.Context, query(c))
			defer release()

			for r := it.Next(); r != nil; r = it.Next() {
				render(c, r)
			}

			return it.Err()
		},
	}
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
