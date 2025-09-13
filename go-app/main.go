package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"go-app/internal/config"
	"go-app/internal/core"
	"go-app/internal/telemetry"
)

func main() {
	// Main context with interrupt signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize telemetry
	tel, shutdown, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		log.Println("Shutting down telemetry...")
		if err := shutdown(ctx); err != nil {
			log.Printf("Error during telemetry shutdown: %v", err)
		}
	}()

	// Create core service
	service := core.NewService(tel)

	// Run the application
	service.Run(ctx)
}
