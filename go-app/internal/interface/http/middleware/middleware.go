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

// responseRecorder captures response status and body.
// This is re-added to fix a compilation error in LoggingMiddleware.
type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// LoggingMiddleware logs incoming requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read and log the request body
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		slog.Info("Incoming request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"body", string(reqBody),
		)

		next.ServeHTTP(rec, r)

		slog.Info("Request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"status", rec.status,
			"response_body", rec.body.String(),
		)
	})
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
