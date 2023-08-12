package system

import (
	"os"

	"golang.org/x/exp/slog"
)

func init() {
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	root := slog.New(h).With("rom", os.Args[0])
	slog.SetDefault(root)
}
