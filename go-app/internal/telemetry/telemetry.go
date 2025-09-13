package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"go-app/internal/config"
)

const (
	TracerName = "main-tracer"
	MeterName  = "main-meter"
)

// Telemetry holds all the telemetry providers
type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
	Tracer         trace.Tracer
	Meter          metric.Meter
	Counter        metric.Int64Counter
}

// Setup initializes all telemetry components
func Setup(ctx context.Context, cfg config.OtelConfig) (*Telemetry, func(context.Context) error, error) {
	// Collect all shutdown functions to execute them in reverse order
	var shutdownFuncs []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var err error
		for i := len(shutdownFuncs) - 1; i >= 0; i-- {
			err = errors.Join(err, shutdownFuncs[i](ctx))
		}
		return err
	}

	// Helper function to handle errors during initialization
	handleErr := func(inErr error) (*Telemetry, func(context.Context) error, error) {
		return nil, shutdown, inErr
	}

	// Create resource that will be used by all telemetry signals
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			// Add more service attributes as needed
			// semconv.ServiceInstanceIDKey.String(generateInstanceID()), // Optional: unique instance ID
		),
	)

	// Add service namespace if provided
	if cfg.ServiceNamespace != "" {
		res, err = resource.Merge(res, resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace),
		))
		if err != nil {
			return handleErr(fmt.Errorf("failed to merge service namespace attribute: %w", err))
		}
	}
	if err != nil {
		return handleErr(fmt.Errorf("failed to create resource: %w", err))
	}

	// Create shared gRPC connection for all exporters
	var creds grpc.DialOption
	if cfg.Insecure {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
		slog.Warn("Using insecure gRPC connection to OTLP collector")
	} else {
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}
	conn, err := grpc.NewClient(cfg.Endpoint, creds)
	if err != nil {
		return handleErr(fmt.Errorf("failed to connect to OTLP endpoint: %w", err))
	}

	// Add connection close to shutdown functions
	shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
		return conn.Close()
	})

	// Initialize provider for each signal
	tp, err := initTracerProvider(ctx, res, conn)
	if err != nil {
		return handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)

	mp, err := initMeterProvider(ctx, res, conn)
	if err != nil {
		return handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)

	lp, err := initLoggerProvider(ctx, res, conn)
	if err != nil {
		return handleErr(err)
	}
	shutdownFuncs = append(shutdownFuncs, lp.Shutdown)

	// Set global providers
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	global.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Bridge OTel logs to slog as default logger
	slog.SetDefault(otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(lp)))

	// Create telemetry instance
	t := &Telemetry{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
		Tracer:         tp.Tracer(TracerName),
		Meter:          mp.Meter(MeterName),
	}

	// Create counter
	t.Counter, err = t.Meter.Int64Counter(
		"app.operations.total",
		metric.WithDescription("Number of operations performed"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create counter: %w", err))
	}

	return t, shutdown, nil
}

// initTracerProvider creates and configures TracerProvider
func initTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	), nil
}

// initMeterProvider creates and configures MeterProvider
func initMeterProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	), nil
}

// initLoggerProvider creates and configures LoggerProvider
func initLoggerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdklog.LoggerProvider, error) {
	exporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}
	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	), nil
}
