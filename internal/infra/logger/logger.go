package logger

import (
	"log/slog"
	"os"
)

func New(logLevel string, addSource, isJSON bool) *slog.Logger {
	var lvl slog.Level

	switch logLevel {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: addSource,
		Level:     slog.Leveler(lvl),
	}
	var handler slog.Handler
	if isJSON == true {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
