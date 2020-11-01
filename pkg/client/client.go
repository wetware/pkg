// Package client provides an API for interacting with live clusters.
package client

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	ww "github.com/wetware/ww/pkg"
	"github.com/wetware/ww/pkg/internal/rpc"
	"github.com/wetware/ww/pkg/internal/rpc/anchor"
	anchorpath "github.com/wetware/ww/pkg/util/anchor/path"
)

// Client interacts with live clusters.  It implements the root Anchor.
type Client struct {
	app *fx.App

	id peer.ID
	ns string

	ps   *topicSet
	term rpc.Terminal
}

// Dial into a cluster using the specified discovery strategy.
// The context is used only to time-out/cancel when dialing into the cluster.
// To terminate the client connection, use the Close method.
func Dial(ctx context.Context, opt ...Option) (c Client, err error) {
	var cfg Config
	for _, f := range withDefault(opt) {
		if err = f(&cfg); err != nil {
			return
		}
	}

	c.app = fx.New(cfg.export(ctx), fx.Populate(&c))
	if err = c.app.Start(ctx); err != nil {
		err = errors.Wrap(err, "dial")
	}

	return
}

// Close the client's cluster connections.
func (c Client) Close() error {
	return c.app.Stop(context.Background())
}

// Loggable fields for Client
func (c Client) Loggable() map[string]interface{} {
	return map[string]interface{}{"ns": c.ns, "id": c.id, "path": "/", "type": "client"}
}

// Join a pubsub topic and returns a Topic handle. Only one Topic handle should
// exist per topic, and Join will error if the Topic handle already exists.
func (c Client) Join(topic string) (Topic, error) {
	return c.ps.Join(topic)
}

// Name of the anchor.  Clients represent the global anchor, so are always named with
// an empty string.
func (c Client) Name() string { return "" }

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

// Load returns a map containing global cluster info
func (c Client) Load(_ context.Context) (ww.Any, error) {
	return nil, errors.New("NOT IMPLEMENTED")
}

// Store is a nop that always returns an error.
func (c Client) Store(context.Context, ww.Any) error {
	return errors.New("not implemented") // store is defined as an error for root anchor
}

// Go is a nop that always returns an error.
func (c Client) Go(context.Context, ...ww.Any) (ww.Any, error) {
	return nil, errors.New("not implemented")
}

/*
	go.uber.org/fx
*/

type clientParams struct {
	fx.In

	Host      host.Host
	Namespace string `name:"ns"`
	PubSub    *pubsub.PubSub
}

func newClient(ctx context.Context, lx fx.Lifecycle, ps clientParams) Client {
	return Client{
		ns:   ps.Namespace,
		id:   ps.Host.ID(),
		term: rpc.NewTerminal(ps.Host),
		ps:   newTopicSet(ps.Namespace, ps.PubSub),
	}
}
