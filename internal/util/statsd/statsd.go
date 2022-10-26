package statsdutil

import (
	"time"

	"github.com/lthibault/log"

	ww "github.com/wetware/ww/pkg"
	"gopkg.in/alexcesaro/statsd.v2"
)

type Env interface {
	Bool(string) bool
	IsSet(string) bool
	String(string) string
	Float64(string) float64
	Duration(string) time.Duration
}

// Metrics wraps a statsd client and satisfies the Wetware
// metrics interface.
type Metrics struct{ *statsd.Client }

// New statsd client.
func New(env Env, log log.Logger) ww.Metrics {
	m, err := statsd.New(
		addr(env),
		muted(env),
		logger(env, log),
		statsd.Prefix("ww"),
		statsd.SampleRate(.1),
		statsd.FlushPeriod(time.Millisecond*250))
	if err != nil {
		log.WithError(err).
			Warn("setup failed for statsd metrics")
		return nopMetrics{}
	}

	return Metrics{m}
}

func (m Metrics) Incr(bucket string) {
	m.Client.Count(bucket, 1)
}

func (m Metrics) Decr(bucket string) {
	m.Client.Count(bucket, -1)
}

func (m Metrics) Duration(bucket string, d time.Duration) {
	m.Client.Timing(bucket, d.Milliseconds())
}

func (m Metrics) WithPrefix(prefix string) ww.Metrics {
	return Metrics{
		Client: m.Client.Clone(statsd.Prefix(prefix)),
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

type nopMetrics struct{}

func (nopMetrics) Incr(string)                    {}
func (nopMetrics) Decr(string)                    {}
func (nopMetrics) Count(string, any)              {}
func (nopMetrics) Gauge(string, any)              {}
func (nopMetrics) Duration(string, time.Duration) {}
func (nopMetrics) Histogram(string, any)          {}
func (nopMetrics) Flush()                         {}
func (nopMetrics) WithPrefix(string) ww.Metrics   { return nopMetrics{} }
