package handler

import (
	"encoding/json"
	"net/http"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/infrastructure/telemetry"
)

// HealthHandler handles requests to the health endpoint
type HealthHandler struct{}

// NewHealthHandler creates a new health handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Handle handles requests to the health endpoint
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create a context with the current request
	ctx := r.Context()

	// Add attributes to the current span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.route", "/health"),
		attribute.String("handler", "health"),
	)

	// Add event to the span
	span.AddEvent("Processing health check")

	// Send log with trace context
	telemetry.Log(ctx, telemetry.LevelInfo, "Processing health check", nil)

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Respond with JSON
	response := map[string]interface{}{
		"status": "healthy",
		"memory": map[string]interface{}{
			"alloc":      m.Alloc,
			"totalAlloc": m.TotalAlloc,
			"sys":        m.Sys,
			"numGC":      m.NumGC,
		},
		"path":   "/health",
		"method": r.Method,
	}

	// Add event to the span
	span.AddEvent("Health check completed successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "health"),
			attribute.String("path", "/health"),
		)
	}
}
