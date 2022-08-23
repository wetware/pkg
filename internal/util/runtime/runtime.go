package runtimeutil

import (
	"context"
	"time"

	"github.com/lthibault/log"
	"github.com/urfave/cli/v2"

	"github.com/wetware/ww/internal/runtime"
	logutil "github.com/wetware/ww/internal/util/log"
	statsdutil "github.com/wetware/ww/internal/util/statsd"
	ww "github.com/wetware/ww/pkg"
)

func New(c *cli.Context) runtime.Env {
	logging := logutil.New(c)
	metrics := statsdutil.New(c, logging)

	return env{
		flags:   c,
		logging: logging,
		metrics: metrics,
	}
}

type env struct {
	flags
	logging log.Logger
	metrics ww.Metrics
}

func (env env) Context() context.Context {
	return env.flags.(*cli.Context).Context
}

func (env env) Log() log.Logger {
	return env.logging
}

func (env env) Metrics() ww.Metrics {
	return env.metrics
}

type flags interface {
	Bool(string) bool
	IsSet(string) bool
	Path(string) string
	String(string) string
	StringSlice(string) []string
	Duration(string) time.Duration
	Float64(string) float64
}
