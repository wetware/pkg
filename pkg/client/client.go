// Package client provides an API for interacting with live clusters.
package client

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"

	log "github.com/lthibault/log/pkg"
	ww "github.com/lthibault/wetware/pkg"
)

// Client interacts with live clusters.
type Client struct {
	log  log.Logger
	host host.Host
	ps   *topicSet
	app  interface{ Stop(context.Context) error }
}

// Dial into a cluster using the specified discovery strategy.  The context is used only
// when dialing into the cluster.  To terminate the client connection, use the Close
// method.
func Dial(ctx context.Context, opt ...Option) (Client, error) {
	var c Client
	app := fx.New(module(&c, opt))
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

// Join a pubsub topic and returns a Topic handle. Only one Topic handle should
// exist per topic, and Join will error if the Topic handle already exists.
func (c Client) Join(topic string) (Topic, error) {
	return c.ps.Join(topic)
}

// Ls the sub-achors
func (c Client) Ls() ww.Iterator {
	panic("Client.Ls NOT IMPLEMENTED")
}

// Walk the Anchor hierarchy.
func (c Client) Walk(ctx context.Context, path []string) ww.Anchor {
	if len(path) == 0 {
		return c
	}

	panic("function NOT IMPLEMENTED")
	// return anchor{id: peer.ID(path[0]), host: c.host}.Walk(ctx, path[1:])
}
