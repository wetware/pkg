package survey

import (
	"context"
	"time"

	"github.com/jpillora/backoff"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/peer"
)

// GradualSurveyor queries a progressively larger subset of the
// cluster keyspace.  It is recommended for bootstrapping large
// clusters where multicast storms may degrade the service.
//
// See specs/survey.md for details.
type GradualSurveyor struct {
	// Factor controls how long the GradualSurveyor will wait for
	// peers at each distance. The raw wait time is calculated by
	// evaluating 'distance * factor', before applying min, max &
	// jitter.  Defaults to 1.5.
	Factor float64

	// Min and Max wait durations.  Defaults to 5s and 90s.
	Min, Max time.Duration

	// If true, use the absolute wait durations calculated via min,
	// max and factor.  By default, the final wait time is sampled
	// uniformly from the interval (w/2, w), where 'w' is the wait
	// duration.
	DisableJitter bool
	*Surveyor
}

func (g GradualSurveyor) FindPeers(ctx context.Context, ns string, opt ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ctxSurv, cancel := context.WithCancel(ctx)

	found, err := g.Surveyor.FindPeers(ctxSurv, ns, append(opt, WithDistance(uint8(0)))...)
	if err != nil {
		cancel()
		return nil, err
	}

	out := make(chan peer.AddrInfo, 1)
	go func() {
		defer close(out)
		defer cancel()

		b := g.backoff()

		for {
			select {
			case <-time.After(b.Duration()):
				cancel()
				ctxSurv, cancel = context.WithCancel(ctx)

				found, err = g.Surveyor.FindPeers(ctxSurv, ns, append(opt, WithDistance(uint8(b.Attempt())))...)
				if err != nil {
					g.sock.Log().Debug("retry failed",
						"error", err)
				}

			case info := <-found:
				select {
				case out <- info:
				default:
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func (g GradualSurveyor) backoff() backoff.Backoff {
	var b = backoff.Backoff{
		Factor: g.Factor,
		Min:    g.Max,
		Max:    g.Max,
		Jitter: !g.DisableJitter,
	}

	if b.Factor == 0 {
		b.Factor = 1.5
	}

	if b.Min == 0 {
		b.Min = time.Second * 5
	}

	if b.Max == 0 {
		b.Max = time.Second * 90
	}

	return b
}
