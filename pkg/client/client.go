// Package client provides an API for interacting with live clusters.
package client

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	host "github.com/libp2p/go-libp2p-core/host"
)

// Client interacts with live clusters.
type Client struct {
	host host.Host
	app  interface{ Stop(context.Context) error }
}

// Dial into a cluster using the specified discovery strategy.  The context is used only
// when dialing into the cluster.  To terminate the client connection, use the Close
// method.
func Dial(ctx context.Context, d Discover, opt ...Option) (Client, error) {
	var c Client
	app := fx.New(module(&c, d, opt))
	c.app = app
	return c, errors.Wrap(app.Start(ctx), "dial")
}

// Close the client's cluster connections.
func (c Client) Close() error {
	return c.app.Stop(context.Background())
}
