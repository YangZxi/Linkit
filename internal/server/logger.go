package server

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(levelStr string) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
