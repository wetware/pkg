package ww

import (
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
	casm "github.com/wetware/casm/pkg"
	protoutil "github.com/wetware/casm/pkg/util/proto"
)

const Version = "0.0.0"

var match = casm.NewMatcher("ww").
	Then(protoutil.SemVer(Version))

// Subprotocol returns a protocol.ID that matches the
// pattern:  /casm/<casm-version>/ww/<version>/<ns>/<...>
func Subprotocol(ns string, ss ...string) protocol.ID {
	return casm.Subprotocol("ww", append([]string{Version, ns}, ss...)...)
}

// NewMatcher returns a stream matcher for a protocol.ID
// that matches the pattern:  /ww/<version>/<ns>
func NewMatcher(ns string) protoutil.MatchFunc {
	return match.Then(protoutil.Exactly(ns))
}

type Metrics interface {
	// Incr is equivalent to Count(bucket, 1).
	Incr(bucket string)

	// Decr is equivalent to Count(bucket, -1).
	Decr(bucket string)

	// Count increments the bucket by number.
	Count(bucket string, number any)

	// Gauge sends the absolute value of number to the given bucket.
	Gauge(bucket string, number any)

	// Duration sends a time interval to the given bucket.
	// Precision is implementation-dependent, but usually
	// on the order of milliseconds.
	Duration(bucket string, d time.Duration)

	// Histogram sends an histogram value to a bucket.
	Histogram(bucket string, value any)

	// WithPrefix returns a new Metrics instance with 'prefix'
	// appended to the current prefix string, separated by '.'.
	WithPrefix(prefix string) Metrics

	// Flush writes all buffered metrics.
	Flush()
}
