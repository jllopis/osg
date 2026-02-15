package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"osg/internal/config"
)

func New(cfg config.LoggingConfig, verbose bool) *slog.Logger {
	return NewWithWriter(cfg, verbose, os.Stdout)
}

func NewWithWriter(cfg config.LoggingConfig, verbose bool, writer io.Writer) *slog.Logger {
	if writer == nil {
		writer = os.Stdout
	}

	level := parseLevel(cfg.Level)
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{Level: level}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))

	// Auto-detect: when format is "json" (the default) but the writer
	// is connected to an interactive terminal, switch to the more
	// human-friendly text format.  Users can still force JSON via
	// logging.format: "json" + piping to a file.
	if format != "text" && IsTTY(writer) {
		format = "text"
	}

	if format == "text" {
		return slog.New(slog.NewTextHandler(writer, opts))
	}

	return slog.New(slog.NewJSONHandler(writer, opts))
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
