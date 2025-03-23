package service

import (
	"context"
	"log/slog"

	"github.com/urfave/cli/v2"
)

type CLI struct {
	App  *cli.App
	Args []string
}

func (c CLI) String() string {
	return c.App.Name
}

func (c CLI) Serve(ctx context.Context) error {
	slog.DebugContext(ctx, "application started",
		"name", c.App.Name,
		"args", c.Args)
	defer slog.DebugContext(ctx, "aplication finished",
		"name", c.App.Name,
		"args", c.Args)

	return c.App.RunContext(ctx, c.Args)
}
