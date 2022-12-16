//go:generate mockgen -source=conn.go -destination=../../internal/mock/pkg/client/conn.go -package=mock_client

package client

import (
	"context"
	"errors"
	"sync"

	"capnproto.org/go/capnp/v3"
)

// ErrDisconnected indicates that the client's connection to
// the cluster was lost.
var ErrDisconnected = errors.New("disconnected")

// Conn is cluster connection.  Implementations MAY cache
// connections to individual hosts.
type Conn interface {
	// Bootstrap returns the remote vat's bootstrap interface.
	// The caller MUST release the client when finished.
	Bootstrap(context.Context) capnp.Client

	// Done returns a read-only channel that is closed when
	// the conn becomes disconnected from the cluster.
	Done() <-chan struct{}

	// Close the cluster connection, releasing all resouces.
	Close() error
}

// HostConn is a connection to an individual host in the cluster.
// It caches the bootstrap client, allowing callers to invoke the
// Bootstrap() method repeatedly.
type HostConn struct {
	Conn

	mu sync.Mutex
	bc capnp.Client
}

// Bootstrap returns the remote vat's bootstrap interface.
// Bootstrap clients are cached until they become invalid.
//
// The caller MUST release the client when finished.
func (c *HostConn) Bootstrap(ctx context.Context) capnp.Client {
	c.mu.Lock()
	defer c.mu.Unlock()

	// null, or previous bootstrap attempt failed?
	if c.invalid() {
		c.bc = c.Conn.Bootstrap(ctx)
	}

	return c.bc
}

func (c *HostConn) invalid() bool {
	if c.bc.IsValid() {
		_, ok := c.bc.State().Brand.Value.(error)
		return ok
	}

	return true
}
