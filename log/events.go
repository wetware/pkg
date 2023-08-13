package log

import (
	"fmt"

	"golang.org/x/exp/slog"
)

type Event struct {
	Level   slog.Level
	Message string
	Args    []any
}

func (ev Event) String() string {
	switch ev.Level {
	case slog.LevelDebug:
		return ev.Debug()

	case slog.LevelInfo:
		return ev.Render()

	case slog.LevelWarn:
		return ev.Warn()

	case slog.LevelError:
		return ev.Error()
	}

	panic(ev) // unreachable
}

// Debug returns a debug string.
func (ev Event) Debug() string {
	return render(ev)
}

// Render returns application output.
func (ev Event) Render() string {
	return render(ev)
}

// Warn returns urgent application output.
func (ev Event) Warn() string {
	return render(ev)
}

// Error returns the cause of an application failure.
func (ev Event) Error() string {
	return render(ev)
}

func render(ev Event) string {
	switch attr := slog.Any(ev.Message, ev.Args); ev.Level {
	case slog.LevelDebug:
		return fmt.Sprintf("[ DEBUG ][ %s ][ %v ]", ev.Message, attr)
	case slog.LevelInfo:
		return fmt.Sprintf("[ INFO ][ %s ][ %v ]", ev.Message, attr)
	case slog.LevelWarn:
		return fmt.Sprintf("[ WARN ][ %s ][ %v ]", ev.Message, attr)
	case slog.LevelError:
		return fmt.Sprintf("[ ERROR ][ %s ][ %v ]", ev.Message, attr)
	}

	panic(ev) // unreachable

}
