//go:generate mockgen -source=clock.go -destination=test/clock.go -package=test_cluster

package cluster

import (
	"context"
	"time"
)

// Clock tracks the passage of time, allowing Router to update its
// internal state and free resources when the clock has stopped.
// Clock's methods MUST be safe for concurrent access.
type Clock interface {
	// Context returns a context that SHALL expire before a call
	// to Stop() returns.
	Context() context.Context

	// Tick returns a channel that receives the current time
	// at regular intervals.  Implementations SHOULD select a
	// tick interval that is smaller than the minimum expected
	// TTL for the cluster. Implementations MUST NOT close the
	// channel returned by Tick until c.Context() has expired.
	// Closing the channel returned by Tick() is OPTIONAL.
	Tick() <-chan time.Time

	// Stop closes the context returned by Context() and frees
	// all resources.  Relay is guaranteed not to call Start()
	// after Stop() returns.  Relay MAY however call Context()
	// after Stop() has returned.
	Stop()
}

type systemClock struct {
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// NewClock with the specified tick interval.
func NewClock(tick time.Duration) Clock {
	ctx, cancel := context.WithCancel(context.Background())
	return &systemClock{
		ctx:    ctx,
		cancel: cancel,
		ticker: time.NewTicker(tick),
	}
}

func (c *systemClock) Context() context.Context {
	return c.ctx
}

func (c *systemClock) Tick() <-chan time.Time {
	return c.ticker.C
}

func (c *systemClock) Stop() {
	defer c.cancel()
	c.ticker.Stop()
}
