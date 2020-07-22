package start

import (
	"context"

	"github.com/urfave/cli/v2"

	log "github.com/lthibault/log/pkg"
	ctxutil "github.com/wetware/ww/internal/util/ctx"
	logutil "github.com/wetware/ww/internal/util/log"

	"github.com/wetware/ww/pkg/host"
	"github.com/wetware/ww/pkg/runtime"
)

var (
	h host.Host
	l log.Logger

	ctx = ctxutil.WithDefaultSignals(context.Background())
)

// Command constructor
func Command() *cli.Command {
	return &cli.Command{
		Name:   "start",
		Usage:  "start a host process",
		Before: setUp(),
		After:  tearDown(),
		Action: run(),
	}
}

func setUp() cli.BeforeFunc {
	return func(c *cli.Context) (err error) {
		if h, err = host.New(); err == nil {
			l = logutil.New(c).WithFields(h.Loggable())
			l.Info("host started")
		}

		return
	}
}

func tearDown() cli.AfterFunc {
	return func(c *cli.Context) error {
		return h.Close()
	}
}

func run() cli.ActionFunc {
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
