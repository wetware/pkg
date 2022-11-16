package statsdutil

import (
	"time"

	"github.com/wetware/casm/pkg/util/metrics"

	"github.com/lthibault/log"

	"gopkg.in/alexcesaro/statsd.v2"
)

type Env interface {
	Bool(string) bool
	IsSet(string) bool
	String(string) string
	Float64(string) float64
	Duration(string) time.Duration
}

// Client wraps a statsd client and satisfies the Wetware
// metrics interface.
type Client struct{ *statsd.Client }

// New statsd client.
func New(env Env, log log.Logger) metrics.Client {
	c, err := statsd.New(
		addr(env),
		muted(env),
		logger(env, log),
		statsd.Prefix("ww"),
		statsd.SampleRate(.1),
		statsd.FlushPeriod(time.Millisecond*250))
	if err != nil {
		log.WithError(err).
			Warn("setup failed for statsd metrics")
		return metrics.NopClient{}
	}

	return Client{c}
}

func (c Client) Incr(bucket string) {
	c.Client.Count(bucket, 1)
}

func (c Client) Decr(bucket string) {
	c.Client.Count(bucket, -1)
}

func (c Client) Count(bucket string, n int) {
	c.Client.Count(bucket, n)
}

func (c Client) Gauge(bucket string, n int) {
	c.Client.Count(bucket, n)
}

func (c Client) Histogram(bucket string, n int) {
	c.Client.Histogram(bucket, n)
}

func (c Client) Duration(bucket string, d time.Duration) {
	c.Client.Timing(bucket, d.Milliseconds())
}

func (c Client) Timing(t0 time.Time) metrics.Timing {
	return metrics.NewTiming(c, t0)
}

func (c Client) WithPrefix(prefix string) metrics.Client {
	return Client{
		Client: c.Client.Clone(statsd.Prefix(prefix)),
	}
}

func addr(env Env) statsd.Option {
	if env.IsSet("statsd") {
		return statsd.Address(env.String("statsd"))
	}

	return statsd.Address(":8125")
}

func logger(env Env, log log.Logger) statsd.Option {
	return statsd.ErrorHandler(func(err error) {
		log.WithError(err).
			WithField("statsd", env.String("statsd-addr")).
			Warn("failed to send metrics")
	})
}

func muted(env Env) statsd.Option {
	return statsd.Mute(!env.IsSet("statsd"))
}
