package usecase

import (
	"context"

	"go-app/internal/domain/service"
	"go-app/internal/infrastructure/telemetry"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// AppUseCase implements the app service use case
type AppUseCase struct {
	telemetry *telemetry.Telemetry
}

// NewAppUseCase creates a new app use case
func NewAppUseCase(tel *telemetry.Telemetry) service.AppService {
	return &AppUseCase{
		telemetry: tel,
	}
}

// GetWelcomeMessage returns a welcome message
func (uc *AppUseCase) GetWelcomeMessage(ctx context.Context) (map[string]interface{}, error) {
	ctx, span := uc.telemetry.Tracer.Start(ctx, "AppUseCase.GetWelcomeMessage")
	defer span.End()

	span.SetAttributes(
		semconv.HTTPRoute("/"),
		attribute.String("handler", "root"),
	)

	span.AddEvent("Processing root request")

	telemetry.Log(ctx, telemetry.LevelInfo, "Handling root request", nil)

	response := map[string]interface{}{
		"message": "Welcome to the OpenTelemetry Go App",
		"path":    "/",
	}

	return response, nil
}
