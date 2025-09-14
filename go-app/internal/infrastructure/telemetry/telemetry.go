package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go-app/internal/infrastructure/config"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
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
	defaultExportInterval = 60 * time.Second
	defaultExportTimeout  = 30 * time.Second
	defaultMaxQueueSize   = 10000
	defaultBatchTimeout   = 5 * time.Second
)

type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
	Tracer         trace.Tracer
	Meter          metric.Meter
	UserCounter    metric.Int64Counter
	LogVerbosity   int
}

func Setup(ctx context.Context, cfg config.OtelConfig) (*Telemetry, func(context.Context) error, error) {
	var shutdowns []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var err error
		for i := len(shutdowns) - 1; i >= 0; i-- {
			err = errors.Join(err, shutdowns[i](ctx))
		}
		return err
	}
	handleErr := func(e error) (*Telemetry, func(context.Context) error, error) {
		return nil, shutdown, e
	}

	// Build resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
			semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace),
		),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create resource: %w", err))
	}

	protocol := cfg.Protocol
	if protocol == "" {
		protocol = "http"
	}
	slog.Info("Using OTLP protocol", "protocol", protocol, "endpoint", cfg.Endpoint)

	var (
		spanExporter sdktrace.SpanExporter
		metricReader sdkmetric.Reader
		logProcessor sdklog.Processor
	)

	// --- Exporter setup ---
	switch protocol {
	case "grpc":
		conn, err := grpc.NewClient(cfg.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			slog.Error("Failed to connect to OTLP gRPC", "endpoint", cfg.Endpoint, "err", err)
			return handleErr(err)
		}
		spanExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("trace exporter gRPC: %w", err))
		}
		metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("metric exporter gRPC: %w", err))
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExp)

		logExp, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
		if err != nil {
			return handleErr(fmt.Errorf("log exporter gRPC: %w", err))
		}
		logProcessor = newBatchProcessor(logExp)

	default: // HTTP
		traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
		logOpts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
			logOpts = append(logOpts, otlploghttp.WithInsecure())
			slog.Warn("Using insecure HTTP connection", "endpoint", cfg.Endpoint)
		}

		spanExporter, err = otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			slog.Warn("OTLP trace exporter unreachable", "endpoint", cfg.Endpoint, "err", err)
			return handleErr(err)
		}

		metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			slog.Warn("OTLP metric exporter unreachable", "endpoint", cfg.Endpoint, "err", err)
			return handleErr(err)
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExp)

		logExp, err := otlploghttp.New(ctx, logOpts...)
		if err != nil {
			slog.Warn("OTLP log exporter unreachable", "endpoint", cfg.Endpoint, "err", err)
			return handleErr(err)
		}
		logProcessor = newBatchProcessor(logExp)
	}

	// --- Providers ---
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter,
			sdktrace.WithMaxQueueSize(defaultMaxQueueSize),
			sdktrace.WithBatchTimeout(defaultBatchTimeout),
			sdktrace.WithExportTimeout(defaultExportTimeout)),
		sdktrace.WithResource(res),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(logProcessor),
		sdklog.WithResource(res),
	)

	// Register shutdowns
	shutdowns = append(shutdowns, tracerProvider.Shutdown, meterProvider.Shutdown, loggerProvider.Shutdown)

	// Set globals
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	global.SetLoggerProvider(loggerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	slog.SetDefault(otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider)))

	if err := runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
		slog.Error("Failed to start runtime metrics", "err", err)
		return handleErr(err)
	}

	meter := meterProvider.Meter(cfg.MeterName)
	userCounter, err := meter.Int64Counter("user_operations_total", metric.WithDescription("Counts user operations"))
	if err != nil {
		return handleErr(fmt.Errorf("user counter: %w", err))
	}

	return &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		LoggerProvider: loggerProvider,
		Tracer:         tracerProvider.Tracer(cfg.TracerName),
		Meter:          meter,
		UserCounter:    userCounter,
		LogVerbosity:   cfg.LogVerbosity,
	}, shutdown, nil
}

func newBatchProcessor(exp sdklog.Exporter) sdklog.Processor {
	return sdklog.NewBatchProcessor(exp,
		sdklog.WithMaxQueueSize(defaultMaxQueueSize),
		sdklog.WithExportInterval(defaultExportInterval),
		sdklog.WithExportTimeout(defaultExportTimeout),
	)
}
