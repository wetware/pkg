// Package client provides an API for interacting with live clusters.
package client

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"go.uber.org/fx"

	log "github.com/lthibault/log/pkg"
	"github.com/lthibault/wetware/internal/api"
	ww "github.com/lthibault/wetware/pkg"
	anchorpath "github.com/lthibault/wetware/pkg/util/anchor/path"
)

func init() { rand.Seed(time.Now().UnixNano()) }

// Client interacts with live clusters.  It implements the root Anchor.
type Client struct {
	log  log.Logger
	term *terminal
	ps   *topicSet
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

// Ls provides a view of all hosts in the cluster.
func (c Client) Ls(ctx context.Context) ww.Iterator {
	var r api.Router
	r.Client = c.term.AutoDial(ctx, ww.ClusterProtocol)

	res, err := r.Ls(ctx, func(p api.Router_ls_Params) error {
		return nil
	}).Struct()
	if err != nil {
		return errIter(err)
	}

	view, err := res.View()
	if err != nil {
		return errIter(err)
	}

	if !view.HasData() || view.Len() == 0 {
		return emptyIterator{}
	}

	return newClusterView(c.term, view)
}

// Walk the Anchor hierarchy.
func (c Client) Walk(ctx context.Context, path []string) (ww.Anchor, error) {
	if anchorpath.Root(path) {
		return c, nil
	}

	id, err := peer.Decode(path[0])
	if err != nil {
		return nil, err
	}

	var a api.Anchor
	a.Client = c.term.Dial(ctx, ww.AnchorProtocol, id)

	res, err := a.Walk(ctx, func(p api.Anchor_walk_Params) error {
		return p.SetPath(anchorpath.Join(path...))
	}).Struct()
	if err != nil {
		return nil, err
	}

	return anchor{res.Anchor()}, nil
}
