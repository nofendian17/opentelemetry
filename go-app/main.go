package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/memory"
	"go-app/internal/infrastructure/telemetry"
	h "go-app/internal/interface/http"
	"go-app/internal/usecase"
)

func main() {
	// Main context with interrupt signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize telemetry
	tel, shutdown, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() {
		// Create a separate context for shutdown with a timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := shutdown(shutdownCtx); err != nil {
			telemetry.Log(context.Background(), telemetry.LevelError, "Error during telemetry shutdown", err)
		}
	}()

	// Create repositories
	userRepo := memory.NewInMemoryRepository()

	// Create use cases
	userUseCase := usecase.NewUserUseCase(userRepo, tel)
	appUseCase := usecase.NewAppUseCase(tel)

	// Create HTTP handler
	handler := h.NewHandler(userUseCase, appUseCase, tel, cfg)

	// Start server in a goroutine
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	go func() {
		fmt.Printf("Server starting on %s\n", cfg.AppPort)
		telemetry.Log(serverCtx, telemetry.LevelInfo, fmt.Sprintf("Starting server on %s", cfg.AppPort), nil)
		if err := handler.StartWithAddr(serverCtx, cfg.AppPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errors.Is(err, syscall.EADDRINUSE) {
				fmt.Fprintf(os.Stderr, "Port %s is already in use. Please choose another port.\n", cfg.AppPort)
				telemetry.Log(serverCtx, telemetry.LevelError, fmt.Sprintf("Port %s is already in use", cfg.AppPort), err)
				os.Exit(1)
			} else {
				telemetry.Log(serverCtx, telemetry.LevelError, "Server failed to start", err)
			}
		}
	}()

	fmt.Println("Server started... Press Ctrl+C to exit.")

	// Wait for interrupt signal
	<-ctx.Done()

	fmt.Println("\nShutting down application gracefully...")
	telemetry.Log(serverCtx, telemetry.LevelInfo, "Shutting down application gracefully", nil)
	// Shutdown HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := handler.Stop(shutdownCtx); err != nil {
		telemetry.Log(shutdownCtx, telemetry.LevelError, "Error during server shutdown", err)
	}
}
