package system

import (
	"log/slog"
	"os"
)

func init() {
	var level = slog.LevelInfo
	switch os.Getenv("WW_LOGLVL") {
	case slog.LevelDebug.String():
		level = slog.LevelDebug
	case slog.LevelWarn.String():
		level = slog.LevelWarn
	case slog.LevelError.String():
		level = slog.LevelError
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	root := slog.New(h).With("rom", os.Args[0])
	slog.SetDefault(root)
}
