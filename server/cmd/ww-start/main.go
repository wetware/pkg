package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/thejerf/suture/v4"
	"github.com/urfave/cli/v2"

	"github.com/wetware/pkg/server"
	"github.com/wetware/pkg/service"
)

var app = &cli.App{
	Name:           server.Name,
	DefaultCommand: server.DefaultCommand,
	Commands:       server.Commands(),
}

func main() {
	root := suture.New("ww/start", suture.Spec{
		// EventHook: ,  // TODO
	})

	root.Add(service.CLI{App: app})

	if err := root.Serve(context.TODO()); err != nil {
		slog.Error("application failed",
			"reason", err)

		var e cli.ExitCoder
		if errors.As(err, &e) {
			os.Exit(e.ExitCode())
		} else {
			// -1 is a generic signal for "unhandled event".
			os.Exit(-1)
		}
	}
}
