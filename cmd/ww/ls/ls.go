package ls

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/libp2p/go-libp2p"
	local "github.com/libp2p/go-libp2p/core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/system"
)

func Command() *cli.Command {
	return &cli.Command{
		Name: "ls",
		Action: func(c *cli.Context) error {
			h, err := clientHost(c)
			if err != nil {
				return err
			}
			defer h.Close()

			host, err := system.Boot[host.Host](c, h)
			if err != nil {
				return err
			}

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

func clientHost(c *cli.Context) (local.Host, error) {
	return libp2p.New(
		libp2p.NoTransports,
		libp2p.NoListenAddrs,
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport))
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
