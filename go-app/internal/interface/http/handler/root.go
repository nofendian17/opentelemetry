package handler

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/domain/service"
	"go-app/internal/infrastructure/telemetry"
)

// RootHandler handles requests to the root endpoint
type RootHandler struct {
	appService service.AppService
}

// NewRootHandler creates a new root handler
func NewRootHandler(appService service.AppService) *RootHandler {
	return &RootHandler{
		appService: appService,
	}
}

// Handle handles requests to the root endpoint
func (h *RootHandler) Handle(w http.ResponseWriter, r *http.Request) {
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
		attribute.String("http.route", "/"),
		attribute.String("handler", "root"),
	)

	// Get welcome message from application service
	response, err := h.appService.GetWelcomeMessage(ctx)
	if err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to get welcome message", err,
			attribute.String("handler", "root"),
			attribute.String("path", "/"),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add request method to response
	response["method"] = r.Method

	// Respond with JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Failed to encode response", err,
			attribute.String("handler", "root"),
			attribute.String("path", "/"),
		)
	}
}
