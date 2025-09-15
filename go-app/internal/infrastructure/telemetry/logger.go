package telemetry

import (
	"context"
	"log/slog"
	"sync"

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

// Global variable to store the current log verbosity level
var (
	logVerbosity int
	mu           sync.RWMutex
)

// SetLogVerbosity sets the global log verbosity level
func SetLogVerbosity(verbosity int) {
	mu.Lock()
	defer mu.Unlock()
	logVerbosity = verbosity
}

// GetLogVerbosity gets the current log verbosity level
func GetLogVerbosity() int {
	mu.RLock()
	defer mu.RUnlock()
	return logVerbosity
}

// shouldLogMessage determines if a message should be logged based on verbosity level
func shouldLogMessage(level LogLevel) bool {
	verbosity := GetLogVerbosity()

	switch level {
	case LevelError:
		// Always log errors
		return true
	case LevelWarn:
		// Log warnings when verbosity is 1 or higher
		return verbosity >= 1
	case LevelInfo:
		// Log info messages only when verbosity is 2 (verbose)
		return verbosity >= 2
	default:
		return verbosity >= 2
	}
}

// Log logs a message with telemetry context at the given level.
// If err is non-nil, it is recorded in the span and logged.
func Log(ctx context.Context, level LogLevel, msg string, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)

	// For performance, only convert attributes when needed
	var logAttrs []any
	shouldLog := shouldLogMessage(level)

	if shouldLog {
		logAttrs = attrsToLogAttrs(attrs)
	}

	switch level {
	case LevelError:
		if span.IsRecording() {
			span.SetStatus(codes.Error, msg)
			if err != nil {
				span.RecordError(err, trace.WithAttributes(attrs...))
			}
		}
		if err != nil && shouldLog {
			logAttrs = append(logAttrs, slog.String("error", err.Error()))
		}
		if shouldLog {
			slog.ErrorContext(ctx, msg, logAttrs...)
		}
	case LevelWarn:
		if span.IsRecording() {
			span.AddEvent(msg, trace.WithAttributes(attrs...))
		}
		if shouldLog {
			slog.WarnContext(ctx, msg, logAttrs...)
		}
	default:
		if span.IsRecording() {
			span.AddEvent(msg, trace.WithAttributes(attrs...))
		}
		if shouldLog {
			slog.InfoContext(ctx, msg, logAttrs...)
		}
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
