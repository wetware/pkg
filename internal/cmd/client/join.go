package client

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli/v2"
	"github.com/wetware/casm/pkg/boot"
	"github.com/wetware/ww/pkg/client"
	"golang.org/x/sync/errgroup"
)

// ww client join /ip4/10.0.1.232/udp/2020/quic
func Join() *cli.Command {
	return &cli.Command{
		Name:      "join",
		Usage:     "merge the local cluster with the cluster(s) containing the muliaddr argument(s)",
		ArgsUsage: "[multiaddrs...]",
		Before:    dial(),
		Action:    join(),
	}
}

func join() cli.ActionFunc {
	return func(c *cli.Context) error {
		remote, err := addresses(c)
		if err != nil {
			return err
		}

		local, err := view(c)
		if err != nil {
			return err
		}

		g, ctx := errgroup.WithContext(c.Context)
		for _, peer := range remote {
			g.Go(introduce(ctx, local, peer))
		}

		return g.Wait()
	}
}

func addresses(c *cli.Context) (boot.StaticAddrs, error) {
	if c.Args().Present() {
		return boot.NewStaticAddrStrings(c.Args().Slice()...)
	}

	return nil, errors.New("must provide at least one addr from remote cluster")
}

func introduce(ctx context.Context, local []client.Anchor, remote peer.AddrInfo) func() error {
	// copy slice for concurrent access
	local = append([]client.Anchor{}, local...)

	return func() (err error) {
		rand.Shuffle(len(local), func(i, j int) {
			local[i], local[j] = local[j], local[i]
		})

		for _, a := range local {
			if err = a.(*client.Host).Join(ctx, remote); err == nil {
				break
			}
		}

		return
	}
}

func view(c *cli.Context) ([]client.Anchor, error) {
	var (
		as []client.Anchor
		it = node.Ls(c.Context)
	)

	for it.Next() {
		as = append(as, it.Anchor())
	}

	if len(as) == 0 {
		return nil, errors.New("no peers in local cluster")
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(as), func(i, j int) {
		as[i], as[j] = as[j], as[i]
	})

	return as, it.Err()
}

func hostpath(info peer.AddrInfo) []string {
	return []string{info.ID.String()}
}
