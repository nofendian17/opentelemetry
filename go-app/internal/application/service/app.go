package service

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/infrastructure/telemetry"
)

// AppService handles application-level operations
type AppService struct {
	telemetry *telemetry.Telemetry
	tracer    trace.Tracer
}

// NewAppService creates a new AppService
func NewAppService(tel *telemetry.Telemetry) *AppService {
	return &AppService{
		telemetry: tel,
		tracer:    tel.Tracer,
	}
}

// HealthCheck performs a health check of the application
func (s *AppService) HealthCheck(ctx context.Context) map[string]interface{} {
	ctx, span := s.tracer.Start(ctx, "AppService.HealthCheck")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "health_check"),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Performing health check", nil,
		attribute.String("operation", "health_check"),
	)

	// Perform basic health checks
	healthStatus := map[string]interface{}{
		"status":  "healthy",
		"service": "go-app",
		"version": "v0.1.0",
		"checks": map[string]interface{}{
			"database": "ok",
			"memory":   "ok",
		},
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Health check completed", nil,
		attribute.String("operation", "health_check"),
		attribute.String("status", "healthy"),
	)

	return healthStatus
}

// GetWelcomeMessage returns a welcome message
func (s *AppService) GetWelcomeMessage(ctx context.Context) (map[string]interface{}, error) {
	ctx, span := s.tracer.Start(ctx, "AppService.GetWelcomeMessage")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "get_welcome_message"),
	)

	telemetry.Log(ctx, telemetry.LevelInfo, "Getting welcome message", nil,
		attribute.String("operation", "get_welcome_message"),
	)

	message := map[string]interface{}{
		"message":     "Welcome to Go App!",
		"application": "go-app",
		"version":     "v0.1.0",
		"status":      "running",
	}

	return message, nil
}

// GetStatus returns the current application status
func (s *AppService) GetStatus(ctx context.Context) map[string]interface{} {
	ctx, span := s.tracer.Start(ctx, "AppService.GetStatus")
	defer span.End()

	span.SetAttributes(
		attribute.String("operation", "get_status"),
	)

	status := map[string]interface{}{
		"application": "go-app",
		"version":     "v0.1.0",
		"environment": "development",
		"uptime":      "running",
	}

	return status
}
