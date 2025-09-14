package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go-app/internal/infrastructure/config"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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
	"google.golang.org/grpc/credentials/insecure"
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
	UserCounter    metric.Int64Counter
}

// Setup initializes all telemetry components
func Setup(ctx context.Context, cfg config.OtelConfig) (*Telemetry, func(context.Context) error, error) {
	// Helper to collect shutdown functions
	var shutdowns []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var err error
		for i := len(shutdowns) - 1; i >= 0; i-- {
			err = errors.Join(err, shutdowns[i](ctx))
		}
		return err
	}

	// Helper for error returns
	handleErr := func(e error) (*Telemetry, func(context.Context) error, error) {
		return nil, shutdown, e
	}

	// Build resource
	resAttrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	}
	if cfg.ServiceNamespace != "" {
		resAttrs = append(resAttrs, semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace))
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(resAttrs...),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create resource: %w", err))
	}

	// Determine OTLP protocol (gRPC or HTTP)
	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	if protocol == "" {
		protocol = "http" // Default to http
	}
	slog.Info("Using OTLP protocol", "protocol", protocol)

	// Create exporters based on protocol
	var spanExporter sdktrace.SpanExporter
	var metricReader sdkmetric.Reader
	var logProcessor sdklog.Processor

	if protocol == "grpc" {
		// gRPC exporter setup
		conn, err := grpc.NewClient(cfg.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return handleErr(fmt.Errorf("failed to create gRPC connection to %s: %w", cfg.Endpoint, err))
		}

		spanExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("failed to create gRPC trace exporter: %w", err))
		}

		metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("failed to create gRPC metric exporter: %w", err))
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExporter)

		logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("failed to create gRPC log exporter: %w", err))
		}
		logProcessor = sdklog.NewBatchProcessor(logExporter)

	} else {
		// HTTP exporter setup
		traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
		logOpts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
			logOpts = append(logOpts, otlploghttp.WithInsecure())
			slog.Warn("Using insecure HTTP connection to OTLP collector")
		}

		traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			return handleErr(fmt.Errorf("failed to create HTTP trace exporter: %w", err))
		}
		spanExporter = traceExporter

		metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			return handleErr(fmt.Errorf("failed to create HTTP metric exporter: %w", err))
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExporter)

		logExporter, err := otlploghttp.New(ctx, logOpts...)
		if err != nil {
			return handleErr(fmt.Errorf("failed to create HTTP log exporter: %w", err))
		}
		logProcessor = sdklog.NewBatchProcessor(logExporter)
	}

	// Create and register providers
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter),
		sdktrace.WithResource(res),
	)
	shutdowns = append(shutdowns, tracerProvider.Shutdown)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	shutdowns = append(shutdowns, meterProvider.Shutdown)

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(logProcessor),
		sdklog.WithResource(res),
	)
	shutdowns = append(shutdowns, loggerProvider.Shutdown)

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	global.SetLoggerProvider(loggerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	slog.SetDefault(otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider)))

	// Start runtime metrics collection
	if err := runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
		return handleErr(fmt.Errorf("failed to start runtime metrics: %w", err))
	}

	// Create custom metrics
	meter := meterProvider.Meter(MeterName)
	userCounter, err := meter.Int64Counter(
		"user_operations_total",
		metric.WithDescription("Counts user operations by type and status"),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create user counter: %w", err))
	}

	t := &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		LoggerProvider: loggerProvider,
		Tracer:         tracerProvider.Tracer(TracerName),
		Meter:          meter,
		UserCounter:    userCounter,
	}
	return t, shutdown, nil
}
