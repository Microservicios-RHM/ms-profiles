package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

// ServiceName coincide con OTEL_SERVICE_NAME, para cruzar en Grafana los logs (Loki) con las trazas
// (Zipkin) del mismo servicio.
func serviceName() string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	return "perfiles-service"
}

// NewLogger produce logs JSON de una línea con los campos mínimos del Reto 8:
//
//	{"timestamp":"...","level":"INFO","service":"perfiles-service","traceId":"...","message":"..."}
//
// traceId/spanId salen del span activo del context.Context: por eso los logs de negocio usan
// InfoContext/WarnContext(ctx, ...).
func NewLogger(level string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: replaceAttr,
	})
	return slog.New(traceHandler{Handler: handler}).With("service", serviceName())
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

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		a.Key = "timestamp"
		a.Value = slog.StringValue(a.Value.Time().UTC().Format("2006-01-02T15:04:05.000Z07:00"))
	case slog.MessageKey:
		a.Key = "message"
	case slog.LevelKey:
		a.Key = "level"
		a.Value = slog.StringValue(strings.ToUpper(a.Value.String()))
	}
	return a
}

// traceHandler agrega traceId y spanId del span activo a cada registro.
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, record slog.Record) error {
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		record.AddAttrs(
			slog.String("traceId", spanContext.TraceID().String()),
			slog.String("spanId", spanContext.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, record)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}
