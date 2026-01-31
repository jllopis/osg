package logging

import (
	"log/slog"
	"os"
	"strings"

	"osg/internal/config"
)

func New(cfg config.LoggingConfig, verbose bool) *slog.Logger {
	level := parseLevel(cfg.Level)
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
