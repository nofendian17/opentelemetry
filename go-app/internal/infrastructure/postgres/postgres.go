package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Client wraps gorm.DB with additional functionality
type Client struct {
	*gorm.DB
	sqlDB  *sql.DB
	tracer trace.Tracer
}

// NewClient creates a new Postgres client with best practices configuration using GORM
func NewClient(ctx context.Context, cfg config.PostgresConfig, tel *telemetry.Telemetry) (*Client, error) {
	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Info)

	// Open GORM connection
	gormDB, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm postgres connection: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Minute)

	// Note: OpenTelemetry tracing plugin removed due to dependency issues
	// Can be added back when dependencies are properly resolved

	// Test the connection
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Successfully connected to Postgres with GORM", nil,
		attribute.String("postgres.dsn", maskDSN(cfg.DSN)),
		attribute.Int("postgres.max_open_conns", cfg.MaxOpenConns),
		attribute.Int("postgres.max_idle_conns", cfg.MaxIdleConns),
	)

	return &Client{
		DB:     gormDB,
		sqlDB:  sqlDB,
		tracer: tel.Tracer,
	}, nil
}

// HealthCheck performs a health check on the Postgres connection
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "postgres.health_check")
	defer span.End()

	if err := c.sqlDB.PingContext(ctx); err != nil {
		span.SetAttributes(attribute.Bool("postgres.healthy", false))
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	span.SetAttributes(attribute.Bool("postgres.healthy", true))
	return nil
}

// GetStats returns database statistics
func (c *Client) GetStats() sql.DBStats {
	return c.sqlDB.Stats()
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.sqlDB.Close()
}

// GetGormDB returns the underlying GORM DB instance
func (c *Client) GetGormDB() *gorm.DB {
	return c.DB
}

// GetSqlDB returns the underlying sql.DB instance for compatibility
func (c *Client) GetSqlDB() *sql.DB {
	return c.sqlDB
}

// AutoMigrate runs auto migration for given models
func (c *Client) AutoMigrate(dst ...interface{}) error {
	_, span := c.tracer.Start(context.Background(), "postgres.auto_migrate")
	defer span.End()

	err := c.DB.AutoMigrate(dst...)
	if err != nil {
		span.SetAttributes(attribute.Bool("db.error", true))
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	span.SetAttributes(attribute.Bool("migration.success", true))
	return nil
}

// Transaction executes a function within a database transaction
func (c *Client) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	ctx, span := c.tracer.Start(ctx, "postgres.transaction")
	defer span.End()

	err := c.DB.WithContext(ctx).Transaction(fn)
	if err != nil {
		span.SetAttributes(attribute.Bool("db.error", true))
		return fmt.Errorf("transaction failed: %w", err)
	}

	span.SetAttributes(attribute.Bool("transaction.success", true))
	return nil
}

// maskDSN masks sensitive information in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking - in production, use a more sophisticated approach
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-7:]
	}
	return "***"
}
