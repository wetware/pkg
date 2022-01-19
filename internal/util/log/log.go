// Package logutil contains shared utilities for configuring loggers from a cli context.
package logutil

import (
	"github.com/lthibault/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	ww "github.com/wetware/ww/pkg"
)

// New logger from a cli context
func New(c *cli.Context) log.Logger {
	if logger := get(c); logger != nil {
		return logger
	}

	return bind(c)
}

// WithLevel returns a log.Option that configures a logger's level.
func WithLevel(c *cli.Context) (opt log.Option) {
	var level = log.FatalLevel
	defer func() {
		opt = log.WithLevel(level)
	}()

	if c.Bool("trace") {
		level = log.TraceLevel
		return
	}

	if c.String("logfmt") == "none" {
		return
	}

	switch c.String("loglvl") {
	case "trace", "t":
		level = log.TraceLevel
	case "debug", "d":
		level = log.DebugLevel
	case "info", "i":
		level = log.InfoLevel
	case "warn", "warning", "w":
		level = log.WarnLevel
	case "error", "err", "e":
		level = log.ErrorLevel
	case "fatal", "f":
		level = log.FatalLevel
	default:
		level = log.InfoLevel
	}

	return
}

// WithFormat returns an option that configures a logger's format.
func WithFormat(c *cli.Context) log.Option {
	var fmt logrus.Formatter

	switch c.String("logfmt") {
	case "none":
	case "json":
		fmt = &logrus.JSONFormatter{PrettyPrint: c.Bool("prettyprint")}
	default:
		fmt = new(logrus.TextFormatter)
	}

	return log.WithFormatter(fmt)
}

func withErrWriter(c *cli.Context) log.Option {
	return log.WithWriter(c.App.ErrWriter)
}

// key with random component to avoid collision
const key = "ww.util.log:Fp+&(<[.~10}>\\>nI!bzeJZX"

// Bind a global logger instance to the CLI context.
// Future calls to New will return this cached logger.
func bind(c *cli.Context) log.Logger {
	logger := log.New(
		WithLevel(c),
		WithFormat(c),
		withErrWriter(c)).
		WithField("version", ww.Version)

	c.App.Metadata[key] = func() log.Logger {
		return logger
	}

	return logger
}

func get(c *cli.Context) log.Logger {
	if logger, ok := c.App.Metadata[key].(func() log.Logger); ok {
		return logger()
	}

	return nil
}
