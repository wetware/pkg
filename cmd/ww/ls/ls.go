package ls

import (
	"fmt"

	"capnproto.org/go/capnp/v3/rpc"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/cluster/routing"
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

			conn, err := dial(c, h)
			if err != nil {
				return err
			}
			defer conn.Close()

			host, err := bootstrap(c, conn)
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

func bootstrap(c *cli.Context, conn *rpc.Conn) (host.Host, error) {
	client := conn.Bootstrap(c.Context)
	return host.Host(client), client.Resolve(c.Context)
}

func dial(c *cli.Context, h local.Host) (*rpc.Conn, error) {
	return client.Dial(c.Context, h, &client.DialConfig{
		Logger:   slog.Default().WithGroup("local"),
		NS:       c.String("ns"),
		Peers:    c.StringSlice("peer"),
		Discover: c.String("discover"),
	})
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
