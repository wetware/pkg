package ls

import (
	"fmt"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/urfave/cli/v2"

	ww "github.com/wetware/pkg"
	"github.com/wetware/pkg/auth"
	"github.com/wetware/pkg/boot"
	"github.com/wetware/pkg/cap/host"
	"github.com/wetware/pkg/cap/view"
	"github.com/wetware/pkg/client"
	"github.com/wetware/pkg/cluster/routing"
	"github.com/wetware/pkg/util/proto"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "ls",
		Action: list,
	}
}

func list(c *cli.Context) error {
	h, err := client.DialP2P()
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer h.Close()

	d, err := boot.DialString(h, c.String("discover"))
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer d.Close()

	ns := boot.Namespace{
		Name:      c.String("ns"),
		Bootstrap: d,
		Ambient:   d,
	}

	boot := client.BootConfig{
		Net:   ns,
		Host:  h,
		Peers: c.StringSlice("peer"),
		RPC:   nil, // client doesn't export a capabiltity (yet)
	}

	// dial into the cluster;  if -dial=false, client is null.
	sess, err := client.Config[host.Host]{
		PeerDialer: boot,
		Auth:       auth.AllowAll[host.Host],
	}.Dial(c.Context, addr(c, h))
	if err != nil {
		return err
	}
	defer sess.Close()

	host := sess.Client()
	defer host.Release()

	view, release := host.View(c.Context)
	defer release()

	it, release := view.Iter(c.Context, query(c))
	defer release()

	for r := it.Next(); r != nil; r = it.Next() {
		render(c, r)
	}

	return it.Err()
}

func addr(c *cli.Context, h local.Host) *client.Addr {
	return &client.Addr{
		Addr: &ww.Addr{
			NS:  c.String("ns"),
			Vat: h.ID(),
		},
		Protos: proto.Namespace(c.String("ns")),
	}
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}
