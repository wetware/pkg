package statsdutil

import (
	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"
	logutil "github.com/wetware/ww/internal/util/log"
	"gopkg.in/alexcesaro/statsd.v2"
)

// New statsd client.
func New(c *cli.Context, log log.Logger) (*statsd.Client, error) {
	return statsd.New(
		addr(c),
		mute(c),
		tags(c),
		tagfmt(c),
		prefix(c),
		sample(c),
		logger(c, log),
		flushInterval(c))
}

func addr(c *cli.Context) statsd.Option {
	return statsd.Address(c.String("statsd-addr"))
}

func mute(c *cli.Context) statsd.Option {
	return statsd.Mute(!c.Bool("statsd"))
}

func prefix(c *cli.Context) statsd.Option {
	return statsd.Prefix("ww.")
}

func logger(c *cli.Context, l log.Logger) statsd.Option {
	if l == nil {
		l = logutil.New(c)
	}

	l = l.WithField("statsd", c.String("statsd-addr"))
	return statsd.ErrorHandler(func(err error) {
		l.Error(err)
	})
}

func tags(c *cli.Context) statsd.Option {
	var tags = []string{"ww"}
	if c.IsSet("statsd-tag") {
		if c.String("ns") != tags[0] {
			tags = append(tags, c.String("ns"))
		}

		tags = append(tags, c.StringSlice("statsd-tag")...)
	}

	return statsd.Tags(tags...)
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

func flushInterval(c *cli.Context) statsd.Option {
	return statsd.FlushPeriod(c.Duration("statsd-flush"))
}

func sample(c *cli.Context) statsd.Option {
	samp := c.Float64("statsd-sample-rate")
	return statsd.SampleRate(float32(samp))
}
