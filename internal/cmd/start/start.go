package start

import (
	"context"

	"github.com/urfave/cli/v2"

	ctxutil "github.com/wetware/ww/internal/util/ctx"
	logutil "github.com/wetware/ww/internal/util/log"
	ww "github.com/wetware/ww/pkg"

	"github.com/wetware/ww/pkg/host"
)

var (
	h      host.Host
	logger ww.Logger
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
		logger = logutil.New(c)

		if h, err = host.New(host.WithLogger(logger)); err == nil {

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
		logger.With(h).Info("host started")
		defer logger.With(h).Warn("host shutting down")

		<-ctxutil.WithDefaultSignals(context.Background()).Done()

		return nil

		// sub, err := h.EventBus().Subscribe([]interface{}{
		// 	new(runtime.Exception),
		// 	new(runtime.EvtServiceStateChanged),
		// })
		// if err != nil {
		// 	return err
		// }

		// go func() {
		// 	<-ctx.Done()
		// 	sub.Close()
		// }()

		// for v := range sub.Out() {
		// 	switch ev := v.(type) {
		// 	case runtime.Exception:
		// 		l.WithFields(ev.Loggable()).Error("runtime error")
		// 	case runtime.EvtServiceStateChanged:
		// 		l.WithFields(ev.Loggable()).Debug(ev.State)
		// 	}
		// }

		// return nil
	}
}
