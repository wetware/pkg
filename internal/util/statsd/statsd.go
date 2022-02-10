package statsdutil

import (
	"github.com/urfave/cli/v2"
	logutil "github.com/wetware/ww/internal/util/log"
	ww "github.com/wetware/ww/pkg"
	"gopkg.in/alexcesaro/statsd.v2"
)

var tags = []string{
	"ww", ww.Version,
}

// Must returns a new statsd client and panics if an error
// is encountered.
func Must(c *cli.Context) *statsd.Client {
	s, err := New(c)
	if err != nil {
		panic(err)
	}
	return s
}

// New statsd client.
func New(c *cli.Context) (*statsd.Client, error) {
	if s := get(c); s != nil {
		return s, nil
	}

	return bind(c)
}

func addr(c *cli.Context) statsd.Option {
	if c.IsSet("statsd") {
		return statsd.Address(c.String("statsd"))
	}

	return statsd.Address(":8125")
}

func logger(c *cli.Context) statsd.Option {
	return statsd.ErrorHandler(func(err error) {
		logutil.New(c).
			WithField("statsd", c.String("statsd-addr")).
			Error(err)
	})
}

func tagfmt(c *cli.Context) statsd.Option {
	var fmt statsd.TagFormat

	switch c.String("statsd-tagfmt") {
	case "influx":
		fmt = statsd.InfluxDB

	case "datadog":
		fmt = statsd.Datadog
	}

	return statsd.TagsFormat(fmt)
}

func sample(c *cli.Context) statsd.Option {
	samp := c.Float64("statsd-sample-rate")
	return statsd.SampleRate(float32(samp))
}

// key with random component to avoid collision
const key = "ww.util.statsd:0U7]3|~FAJOM#;jXWbA&Gxby"

// Bind a global logger instance to the CLI context.
// Future calls to New will return this cached logger.
func bind(c *cli.Context) (*statsd.Client, error) {
	s, err := statsd.New(
		addr(c),
		tagfmt(c),
		sample(c),
		logger(c),
		statsd.Prefix("ww."),
		statsd.Tags(tags...),
		statsd.Mute(!c.IsSet("statsd")),
		statsd.FlushPeriod(c.Duration("statsd-flush")))

	if err == nil {
		c.App.Metadata[key] = func() *statsd.Client {
			return s
		}
	}

	return s, err
}

func get(c *cli.Context) *statsd.Client {
	if statsd, ok := c.App.Metadata[key].(func() *statsd.Client); ok {
		return statsd()
	}

	return nil
}
