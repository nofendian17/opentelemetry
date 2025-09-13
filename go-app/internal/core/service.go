package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"go-app/internal/telemetry"
)

const (
	TickerDuration   = 1 * time.Second
	OperationTimeout = 4 * time.Second
)

// Service holds the core service dependencies
type Service struct {
	Telemetry *telemetry.Telemetry
}

// NewService creates a new core service
func NewService(tel *telemetry.Telemetry) *Service {
	return &Service{
		Telemetry: tel,
	}
}

// Run executes the main application loop
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(TickerDuration)
	defer ticker.Stop()

	fmt.Println("Telemetry started... Press Ctrl+C to exit.")

	for {
		select {
		case <-ticker.C:
			s.performOperation(ctx)
		case <-ctx.Done():
			fmt.Println("\nShutting down application gracefully...")
			return
		}
	}
}

// performOperation executes a single operation with telemetry
func (s *Service) performOperation(ctx context.Context) {
	// Create a new context with timeout for each operation
	opCtx, cancel := context.WithTimeout(ctx, OperationTimeout)
	defer cancel()

	// Start a new span
	spanCtx, span := s.Telemetry.Tracer.Start(opCtx, "main-operation",
		trace.WithAttributes(attribute.Bool("operation.manual", true)))
	defer span.End()

	span.AddEvent("Starting work")

	// Increment counter
	s.Telemetry.Counter.Add(spanCtx, 1, metric.WithAttributes(attribute.String("operation.result", "success")))

	// Send log that will automatically have TraceID and SpanID from spanCtx
	slog.InfoContext(spanCtx, "Performing operation", slog.String("operation.task", "data-processing"))

	fmt.Printf("Telemetry sent. TraceID=%s\n", span.SpanContext().TraceID().String())

	span.AddEvent("Work completed")
}
