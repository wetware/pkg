// Package client provides an API for interacting with live clusters.
package client

import (
	"context"

	log "github.com/lthibault/log/pkg"
	"github.com/pkg/errors"
	"go.uber.org/fx"

	ww "github.com/lthibault/wetware/pkg"
	"github.com/lthibault/wetware/pkg/internal/rpc"
	"github.com/lthibault/wetware/pkg/internal/rpc/anchor"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

// Client interacts with live clusters.  It implements the root Anchor.
type Client struct {
	log  log.Logger
	ps   *topicSet
	term rpc.Terminal
	app  interface{ Stop(context.Context) error }
}

// Dial into a cluster using the specified discovery strategy.
// The context is used only to time-out/cancel when dialing into the cluster.
// To terminate the client connection, use the Close method.
func Dial(ctx context.Context, opt ...Option) (Client, error) {
	var c Client
	app := fx.New(module(ctx, &c, opt))
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

// String representation of the Client's anchor name.  This always returns "/", but is
// required in order for Client to implement ww.Anchor.
func (c Client) String() string {
	return ""
}

// Path slice.  Required for Client to implement ww.Anchor.
func (c Client) Path() []string {
	return []string{}
}

// Ls provides a view of all hosts in the cluster.
func (c Client) Ls(ctx context.Context) ([]ww.Anchor, error) {
	return anchor.Ls(ctx, c.term, rpc.AutoDial{})
}

// Walk the Anchor hierarchy.
func (c Client) Walk(ctx context.Context, path []string) ww.Anchor {
	if anchorpath.Root(path) {
		return c
	}

	return anchor.Walk(ctx, c.term, rpc.DialString(path[0]), path)
}
