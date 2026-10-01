package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger produce logs JSON de una línea, equivalentes en forma a Pino (ms-employees) y al
// logger de ms-notifications: level, time, service, msg + campos.
func NewLogger(level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: replaceAttr,
	})
	return slog.New(handler).With("service", "perfiles-service")
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
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

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.MessageKey:
		a.Key = "msg"
	case slog.LevelKey:
		a.Key = "level"
		a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
	}
	return a
}
