// Package client provides an API for interacting with live clusters.
package client

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	log "github.com/lthibault/log/pkg"
	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/boot"
)

// Client interacts with live clusters.
type Client struct {
	log  log.Logger
	host host.Host
	app  interface{ Stop(context.Context) error }
}

// Dial into a cluster using the specified discovery strategy.  The context is used only
// when dialing into the cluster.  To terminate the client connection, use the Close
// method.
func Dial(ctx context.Context, b boot.Strategy, opt ...Option) (Client, error) {
	var c Client
	app := fx.New(module(&c, b, opt))
	c.app = app
	return c, errors.Wrap(app.Start(ctx), "dial")
}

// Close the client's cluster connections.
func (c Client) Close() error {
	return c.app.Stop(context.Background())
}

// Log returns a structured logger whose fields identify the client.
func (c Client) Log() log.Logger {
	return c.log
}

// Ls the sub-achors
func (c Client) Ls() ww.Iterator {
	return &rootIterator{
		host: c.host,
		idx:  -1,
		ids:  filterLocalPeer(c.host.ID(), c.host.Network().Peers()),
	}
}

// Walk the Anchor hierarchy.
func (c Client) Walk(ctx context.Context, path []string) ww.Anchor {
	if len(path) == 0 {
		return c
	}

	return anchor{id: peer.ID(path[0]), host: c.host}.Walk(ctx, path[1:])
}

func filterLocalPeer(local peer.ID, ps []peer.ID) (remote []peer.ID) {
	remote = ps[:0] // zero-alloc filtering
	for _, p := range ps {
		if p == local {
			continue
		}

		remote = append(remote, p)
	}

	return
}

type rootIterator struct {
	host host.Host
	idx  int
	ids  []peer.ID
}

func (it *rootIterator) Err() error {
	return nil
}

func (it *rootIterator) Path() string {
	return it.ids[it.idx].String()
}

func (it *rootIterator) Next() (more bool) {

	if more = it.idx < len(it.ids)-1; more {
		it.idx++
	}

	return
}

func (it *rootIterator) Anchor() ww.Anchor {
	return &anchor{id: it.ids[it.idx], host: it.host}
}
