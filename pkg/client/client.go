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
	host    host.Host
	stopper interface{ Stop(context.Context) error }
}

// Dial into a cluster using the specified discovery strategy.  The context is used only
// when dialing into the cluster.  To terminate the client connection, use the Close
// method.
func Dial(ctx context.Context, d Discover) (*Client, error) {
	var host = new(hostWrapper)

	app := fx.New(module(d, &host))
	return &Client{
		host:    host,
		stopper: app,
	}, errors.Wrap(app.Start(ctx), "dial")
}

// Close the client's cluster connections.
func (c Client) Close() error {
	return c.stopper.Stop(context.Background())
}

// `discover` must be a bootstrapper constructor.
// `populate` be pointers.
func module(d Discover, populate ...interface{}) fx.Option {
	return fx.Options(
		fx.NopLogger,
		fx.Supply(d),
		fx.Provide(newBaseContext),
		fx.Provide(newHost),
		fx.Populate(populate...),
	)
}
