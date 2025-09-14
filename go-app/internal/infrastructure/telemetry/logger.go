package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LogLevel string

const (
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Log logs a message with telemetry context at the given level.
// If err is non-nil, it is recorded in the span and logged.
func Log(ctx context.Context, level LogLevel, msg string, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	logAttrs := attrsToLogAttrs(attrs)

	switch level {
	case LevelError:
		if span.IsRecording() {
			span.SetStatus(codes.Error, msg)
			if err != nil {
				span.RecordError(err, trace.WithAttributes(attrs...))
			}
		}
		if err != nil {
			logAttrs = append(logAttrs, slog.String("error", err.Error()))
		}
		slog.ErrorContext(ctx, msg, logAttrs...)
	case LevelWarn:
		if span.IsRecording() {
			span.AddEvent(msg, trace.WithAttributes(attrs...))
		}
		slog.WarnContext(ctx, msg, logAttrs...)
	default:
		if span.IsRecording() {
			span.AddEvent(msg, trace.WithAttributes(attrs...))
		}
		slog.InfoContext(ctx, msg, logAttrs...)
	}
}

// attrsToLogAttrs converts OTel attributes to slog attributes
func attrsToLogAttrs(attrs []attribute.KeyValue) []any {
	logAttrs := make([]any, len(attrs))
	for i, attr := range attrs {
		logAttrs[i] = slog.Any(string(attr.Key), attr.Value.AsInterface())
	}
	return logAttrs
}
