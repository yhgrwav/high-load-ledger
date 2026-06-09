package logger

import (
	"log/slog"
	"os"
)

func New(env, logLevel string, addSource, isJSON *bool) *slog.Logger {
	var lvl slog.Level

	finalLogLevel := "info"
	finalIsJSON := true
	finalAddSource := false

	if env == "development" {
		finalLogLevel = "debug"
		finalIsJSON = false
		finalAddSource = true
	}

	if logLevel != "" {
		finalLogLevel = logLevel
	}
	if isJSON != nil {
		finalIsJSON = *isJSON
	}
	if addSource != nil {
		finalAddSource = *addSource
	}

	switch finalLogLevel {
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
		AddSource: finalAddSource,
		Level:     slog.Leveler(lvl),
	}
	var handler slog.Handler
	if finalIsJSON {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
