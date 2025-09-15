package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/telemetry"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Client wraps sql.DB with additional functionality
type Client struct {
	*sql.DB
	tracer trace.Tracer
}

// NewClient creates a new Postgres client with best practices configuration
func NewClient(ctx context.Context, cfg config.PostgresConfig, tel *telemetry.Telemetry) (*Client, error) {
	// Open database connection with OpenTelemetry tracing
	db, err := otelsql.Open("pgx", cfg.DSN,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			Ping:     true,
			RowsNext: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Minute)

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Successfully connected to Postgres", nil,
		attribute.String("postgres.dsn", maskDSN(cfg.DSN)),
		attribute.Int("postgres.max_open_conns", cfg.MaxOpenConns),
		attribute.Int("postgres.max_idle_conns", cfg.MaxIdleConns),
	)

	return &Client{
		DB:     db,
		tracer: tel.Tracer,
	}, nil
}

// HealthCheck performs a health check on the Postgres connection
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "postgres.health_check")
	defer span.End()

	if err := c.PingContext(ctx); err != nil {
		span.SetAttributes(attribute.Bool("postgres.healthy", false))
		return fmt.Errorf("postgres health check failed: %w", err)
	}

	span.SetAttributes(attribute.Bool("postgres.healthy", true))
	return nil
}

// GetStats returns database statistics
func (c *Client) GetStats() sql.DBStats {
	return c.Stats()
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.DB.Close()
}

// ExecWithTracing executes a query with tracing
func (c *Client) ExecWithTracing(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := c.tracer.Start(ctx, "postgres.exec")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.statement", query),
		attribute.String("db.operation", "exec"),
	)

	result, err := c.ExecContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("db.error", true))
	}

	return result, err
}

// QueryWithTracing queries with tracing
func (c *Client) QueryWithTracing(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := c.tracer.Start(ctx, "postgres.query")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.statement", query),
		attribute.String("db.operation", "query"),
	)

	rows, err := c.QueryContext(ctx, query, args...)
	if err != nil {
		span.SetAttributes(attribute.Bool("db.error", true))
	}

	return rows, err
}

// QueryRowWithTracing queries a single row with tracing
func (c *Client) QueryRowWithTracing(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, span := c.tracer.Start(ctx, "postgres.query_row")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.statement", query),
		attribute.String("db.operation", "query_row"),
	)

	return c.QueryRowContext(ctx, query, args...)
}

// BeginTxWithTracing begins a transaction with tracing
func (c *Client) BeginTxWithTracing(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	ctx, span := c.tracer.Start(ctx, "postgres.begin_tx")
	defer span.End()

	span.SetAttributes(attribute.String("db.operation", "begin_tx"))

	tx, err := c.BeginTx(ctx, opts)
	if err != nil {
		span.SetAttributes(attribute.Bool("db.error", true))
	}

	return tx, err
}

// maskDSN masks sensitive information in DSN for logging
func maskDSN(dsn string) string {
	// Simple masking - in production, use a more sophisticated approach
	if len(dsn) > 20 {
		return dsn[:10] + "***" + dsn[len(dsn)-7:]
	}
	return "***"
}
