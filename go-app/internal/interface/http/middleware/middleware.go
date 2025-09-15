package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Middleware represents a middleware function
type Middleware func(http.Handler) http.Handler

// loggingMiddleware holds the configuration for the logging middleware
type loggingMiddleware struct {
	logBodies bool
}

// LoggingMiddlewareWithConfig creates a logging middleware with the specified config
func LoggingMiddlewareWithConfig(logBodies bool) Middleware {
	return func(next http.Handler) http.Handler {
		lm := &loggingMiddleware{
			logBodies: logBodies,
		}
		return lm.middleware(next)
	}
}

func (lm *loggingMiddleware) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request headers
		headers := make(map[string]string)
		for name, values := range r.Header {
			// Only log the first value for each header
			if len(values) > 0 {
				headers[name] = values[0]
			}
		}

		// Only log request body for non-GET requests and when content length is reasonable
		var reqBody []byte
		if lm.logBodies && r.Body != nil && r.ContentLength > 0 && r.ContentLength < 1024 && r.Method != http.MethodGet {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// Skip body buffering for large responses or when body logging is disabled
		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
			skipBody:       !lm.logBodies || r.ContentLength > 1024,
		}

		// Log request with conditional body logging
		if len(reqBody) > 0 {
			slog.Info("Incoming request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"headers", headers,
				"content_length", r.ContentLength,
				"body", string(reqBody),
			)
		} else {
			slog.Info("Incoming request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"headers", headers,
				"content_length", r.ContentLength,
			)
		}

		next.ServeHTTP(rec, r)

		// Only log response body when it's reasonably small and body logging is enabled
		duration := time.Since(start)
		if lm.logBodies && !rec.skipBody && rec.body.Len() > 0 {
			slog.Info("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", duration,
				"status", rec.status,
				"response_size", rec.body.Len(),
				"response_body", rec.body.String(),
			)
		} else {
			slog.Info("Request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", duration,
				"status", rec.status,
				"response_size", rec.body.Len(),
			)
		}
	})
}

// responseRecorder captures response status and body.
// This is re-added to fix a compilation error in LoggingMiddleware.
type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
	// skipBody controls whether we buffer the response body
	skipBody bool
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	// Only buffer response body if it's reasonably small and body logging is enabled
	if !r.skipBody && len(b) < 1024 {
		r.body.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

// OtelHttpMiddleware adds OpenTelemetry tracing and metrics to requests.
// It uses the standard otelhttp handler, which automatically records
// HTTP server metrics (e.g., duration, request/response size) and creates spans for traces.
func OtelHttpMiddleware(operation string) Middleware {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(
			next,
			operation, // This becomes the span name for the server request
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		)
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Create a context with the current request
				ctx := r.Context()

				// Log the panic
				slog.ErrorContext(ctx, "Panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)

				// Add error to the current span
				span := trace.SpanFromContext(ctx)
				if span.IsRecording() {
					span.SetStatus(500, "Internal Server Error")
					span.RecordError(err.(error), trace.WithAttributes(
						attribute.String("panic", "recovered"),
					))
				}

				// Send error response
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware adds CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(mw ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		if len(mw) == 0 {
			return final
		}

		// Apply middleware in reverse order
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}

		return final
	}
}
