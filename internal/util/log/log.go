// Package logutil contains shared utilities for configuring loggers from a cli context.
package logutil

import (
	"os"

	"github.com/lthibault/log"
	"github.com/sirupsen/logrus"
	"github.com/wetware/pkg"
)

type Env interface {
	Bool(string) bool
	String(string) string
}

// New logger from a cli context
func New(env Env) log.Logger {
	return log.New(
		WithLevel(env),
		WithFormat(env),
		withErrWriter(env)).
		WithField("version", ww.Version)
}

// WithLevel returns a log.Option that configures a logger's level.
func WithLevel(env Env) (opt log.Option) {
	var level = log.FatalLevel
	defer func() {
		opt = log.WithLevel(level)
	}()

	if env.Bool("trace") {
		level = log.TraceLevel
		return
	}

	if env.String("logfmt") == "none" {
		return
	}

	switch env.String("loglvl") {
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
func WithFormat(env Env) log.Option {
	var fmt logrus.Formatter

	switch env.String("logfmt") {
	case "none":
	case "json":
		fmt = &logrus.JSONFormatter{PrettyPrint: env.Bool("prettyprint")}
	default:
		fmt = new(logrus.TextFormatter)
	}

	return log.WithFormatter(fmt)
}

func withErrWriter(env Env) log.Option {
	return log.WithWriter(os.Stderr)
}
