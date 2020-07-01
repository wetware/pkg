package start

import (
	"context"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	log "github.com/lthibault/log/pkg"

	logutil "github.com/lthibault/wetware/internal/util/log"
	runtimeutil "github.com/lthibault/wetware/pkg/util/runtime"

	"github.com/lthibault/wetware/pkg/runtime"
	"github.com/lthibault/wetware/pkg/server"
)

var (
	logger log.Logger
	host   server.Host
)

// Command constructor
func Command(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Before: before(ctx),
		Action: run(ctx),
	}
}

func before(ctx context.Context) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		if host, err = server.New(); err == nil {
			logger = logutil.New(c).WithFields(host.Loggable())
		}

		return
	}
}

func run(ctx context.Context) cli.ActionFunc {
	return func(c *cli.Context) error {
		if err := host.Start(); err != nil {
			return errors.Wrap(err, "start host")
		}

		if err := eventloop(ctx); err != nil {
			return err
		}

		if err := host.Close(); err != nil {
			return errors.Wrap(err, "stop host")
		}

		return nil
	}
}

func eventloop(ctx context.Context) error {
	logger.Info("host started")
	defer logger.Warn("host shutting down")

	stream, err := runtimeutil.CoreEventStream(ctx, host)
	if err != nil {
		return err
	}

	for v := range stream {
		switch ev := v.(type) {
		case runtime.Exception:
			logger.WithFields(ev.Loggable()).Error("runtime error")
		case runtime.EvtServiceStateChanged:
			logger.WithFields(ev.Loggable()).Debug(ev.State)
		}
	}

	return nil
}
