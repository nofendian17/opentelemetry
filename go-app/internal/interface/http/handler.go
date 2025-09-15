package http

import (
	"context"
	"fmt"
	"net/http"

	"go-app/internal/domain/service"
	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/telemetry"
	"go-app/internal/interface/http/middleware"
	"go-app/internal/interface/http/routes"
	"go-app/internal/usecase"
)

// Handler holds the HTTP handler dependencies
type Handler struct {
	userService *usecase.UserUseCase
	appService  service.AppService
	server      *http.Server
	telemetry   *telemetry.Telemetry
	config      config.OtelConfig
}

// NewHandler creates a new HTTP handler
func NewHandler(userService *usecase.UserUseCase, appService service.AppService, tel *telemetry.Telemetry, cfg config.OtelConfig) *Handler {
	return &Handler{
		userService: userService,
		appService:  appService,
		telemetry:   tel,
		config:      cfg,
	}
}

// SetupRoutes sets up the HTTP routes with middleware
func (h *Handler) SetupRoutes() http.Handler {
	// Create a new ServeMux
	mux := http.NewServeMux()

	// Create router and register routes
	router := routes.NewRouter(h.userService, h.appService)
	router.RegisterRoutes(mux)

	// Create middleware chain with config
	middlewareChain := middleware.ChainMiddleware(
		middleware.LoggingMiddlewareWithConfig(h.config.LogBodies),
		middleware.OtelHttpMiddleware("http.server"), // Replaces both tracing and the old metrics middleware
		middleware.RecoveryMiddleware,
		middleware.CORSMiddleware,
	)

	// Apply middleware to the mux
	return middlewareChain(mux)
}

// StartWithAddr Start starts the HTTP server
func (h *Handler) StartWithAddr(ctx context.Context, addr string) error {
	// Set up HTTP routes with middleware
	handler := h.SetupRoutes()

	// Create HTTP server
	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", addr),
		Handler: handler,
	}

	return h.server.ListenAndServe()
}

// Stop stops the HTTP server
func (h *Handler) Stop(ctx context.Context) error {
	if h.server != nil {
		return h.server.Shutdown(ctx)
	}
	return nil
}
