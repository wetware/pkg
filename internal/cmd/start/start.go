package start

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	log "github.com/lthibault/log/pkg"
	logutil "github.com/lthibault/wetware/internal/util/log"

	"github.com/lthibault/wetware/pkg/host"
	"github.com/lthibault/wetware/pkg/runtime"
)

var (
	h host.Host
	l log.Logger
)

// Command constructor
func Command(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Before: setUp(ctx),
		After:  tearDown(),
		Action: run(ctx),
	}
}

func setUp(ctx context.Context) cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()

		if h, err = host.New(ctx); err == nil {
			l = logutil.New(c).WithFields(h.Loggable())
			l.Info("host started")
		}

		return
	}
}

func tearDown() cli.AfterFunc {
	return func(c *cli.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		return h.Shutdown(ctx)
	}
}

func run(ctx context.Context) cli.ActionFunc {
	return func(c *cli.Context) error {
		defer l.Warn("host shutting down")

		sub, err := h.EventBus().Subscribe([]interface{}{
			new(runtime.Exception),
			new(runtime.EvtServiceStateChanged),
		})
		if err != nil {
			return err
		}

		go func() {
			<-ctx.Done()
			sub.Close()
		}()

		for v := range sub.Out() {
			switch ev := v.(type) {
			case runtime.Exception:
				l.WithFields(ev.Loggable()).Error("runtime error")
			case runtime.EvtServiceStateChanged:
				l.WithFields(ev.Loggable()).Debug(ev.State)
			}
		}

		return nil
	}
}
