package ls

import (
	"fmt"

	local "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
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

	d, err := newDiscovery(c, h)
	if err != nil {
		return fmt.Errorf("discovery: %w", err)
	}
	defer d.Close()

	boot := client.BootConfig{
		NS:         c.String("ns"),
		Host:       h,
		Discoverer: d,
	}

	// dial into the cluster
	dialer := client.Dialer[host.Host]{
		Bootstrapper: boot,
		Auth:         auth.AllowAll[host.Host],
		Opts:         nil, // TODO:  export something from the client side
	}

	sess, err := dialer.Dial(c.Context, addr(c, h))
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

func addr(c *cli.Context, h local.Host) *ww.Addr {
	return &ww.Addr{
		NS:    c.String("ns"),
		Peer:  h.ID(),
		Proto: proto.Namespace(c.String("ns")),
	}
}

func query(c *cli.Context) view.Query {
	return view.NewQuery(view.All())
}

func render(c *cli.Context, r routing.Record) {
	fmt.Fprintf(c.App.Writer, "/%s\n", r.Server())
}

func newDiscovery(c *cli.Context, h local.Host) (_ boot.Service, err error) {
	// use discovery service?
	if len(c.StringSlice("peer")) == 0 {
		serviceAddr := c.String("discover")
		return boot.DialString(h, serviceAddr)
	}

	// fast path; direct dial a peer
	maddrs := make([]ma.Multiaddr, len(c.StringSlice("peer")))
	for i, s := range c.StringSlice("peer") {
		if maddrs[i], err = ma.NewMultiaddr(s); err != nil {
			return
		}
	}

	infos, err := peer.AddrInfosFromP2pAddrs(maddrs...)
	return boot.StaticAddrs(infos), err
}
