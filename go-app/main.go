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

	"go-app/internal/application/service"
	"go-app/internal/application/worker"
	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/kafka"
	"go-app/internal/infrastructure/postgres"
	"go-app/internal/infrastructure/redis"
	postgresrepo "go-app/internal/infrastructure/repository/postgres"
	"go-app/internal/infrastructure/telemetry"
	h "go-app/internal/interface/http"
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

	// Create postgres client
	pgDB, err := postgres.NewClient(ctx, cfg.Postgres, tel)
	if err != nil {
		log.Fatalf("Failed to initialize postgres: %v", err)
	}
	defer func() {
		if err := pgDB.Close(); err != nil {
			telemetry.Log(context.Background(), telemetry.LevelError, "Error during postgres shutdown", err)
		}
	}()

	// Create redis client
	rdb, err := redis.NewClient(ctx, cfg.Redis, tel)
	if err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			telemetry.Log(context.Background(), telemetry.LevelError, "Error during redis shutdown", err)
		}
	}()

	// Create kafka producer
	kproducer, err := kafka.NewProducer(cfg.Kafka, tel)
	if err != nil {
		log.Fatalf("Failed to initialize kafka producer: %v", err)
	}
	defer kproducer.Close()

	// Create kafka consumer
	kconsumer, err := kafka.NewConsumer(cfg.Kafka, cfg.Kafka.ConsumerGroup, tel)
	if err != nil {
		log.Fatalf("Failed to initialize kafka consumer: %v", err)
	}
	defer kconsumer.Close()

	// Create and start Kafka worker
	kafkaWorker := worker.NewKafkaWorker(kconsumer, tel)
	kafkaWorker.Start(ctx)

	// Auto-migration will handle table creation and updates
	if err := pgDB.AutoMigrate(&postgresrepo.UserModel{}); err != nil {
		log.Fatalf("Failed to run auto migration: %v", err)
	}

	// Create repositories using GORM DB
	userRepo := postgresrepo.NewPostgresUserRepository(pgDB.GetGormDB())

	// Create services
	userService := service.NewUserService(userRepo, tel)
	appService := service.NewAppService(tel)

	// Create HTTP handler
	handler := h.NewHandler(userService, appService, tel, cfg.Otel)

	// Start server in a goroutine
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	go func() {
		fmt.Printf("Server starting on %s\n", cfg.Otel.AppPort)
		telemetry.Log(serverCtx, telemetry.LevelInfo, fmt.Sprintf("Starting server on %s", cfg.Otel.AppPort), nil)
		if err := handler.StartWithAddr(serverCtx, cfg.Otel.AppPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if errors.Is(err, syscall.EADDRINUSE) {
				fmt.Fprintf(os.Stderr, "Port %s is already in use. Please choose another port.\n", cfg.Otel.AppPort)
				telemetry.Log(serverCtx, telemetry.LevelError, fmt.Sprintf("Port %s is already in use", cfg.Otel.AppPort), err)
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
