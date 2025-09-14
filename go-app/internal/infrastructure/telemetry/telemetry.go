package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
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

	"go-app/internal/infrastructure/config"
)

const (
	TracerName = "main-tracer"
	MeterName  = "main-meter"
)

// Telemetry holds all the telemetry providers
type Telemetry struct {
	TracerProvider  *sdktrace.TracerProvider
	MeterProvider   *sdkmetric.MeterProvider
	LoggerProvider  *sdklog.LoggerProvider
	Tracer          trace.Tracer
	Meter           metric.Meter
	UserCounter     metric.Int64Counter
	RequestDuration metric.Float64Histogram
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
	hostname, _ := os.Hostname()
	resAttrs := []attribute.KeyValue{
		semconv.HostNameKey.String(hostname),
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	}
	res, err := resource.New(ctx, resource.WithAttributes(resAttrs...))
	if err != nil {
		return handleErr(fmt.Errorf("failed to create resource: %w", err))
	}
	if cfg.ServiceNamespace != "" {
		res, err = resource.Merge(res, resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNamespaceKey.String(cfg.ServiceNamespace),
		))
		if err != nil {
			return handleErr(fmt.Errorf("failed to merge service namespace: %w", err))
		}
	}

	// HTTP options builder
	traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
	logOpts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		logOpts = append(logOpts, otlploghttp.WithInsecure())
		slog.Warn("Using insecure HTTP connection to OTLP collector")
	}

	// Providers
	tp, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create trace exporter: %w", err))
	}
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(tp),
		sdktrace.WithResource(res),
	)
	shutdowns = append(shutdowns, tracerProvider.Shutdown)

	mp, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create metric exporter: %w", err))
	}
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(mp)),
		sdkmetric.WithResource(res),
	)
	shutdowns = append(shutdowns, meterProvider.Shutdown)

	lp, err := otlploghttp.New(ctx, logOpts...)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create log exporter: %w", err))
	}
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(lp)),
		sdklog.WithResource(res),
	)
	shutdowns = append(shutdowns, loggerProvider.Shutdown)

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	global.SetLoggerProvider(loggerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	slog.SetDefault(otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider)))

	userCounter, err := meterProvider.Meter(MeterName).Int64Counter(
		"user_operations_total",
		metric.WithDescription("Counts user operations by type and status"),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create user counter: %w", err))
	}

	requestDuration, err := meterProvider.Meter(MeterName).Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create request duration histogram: %w", err))
	}

	t := &Telemetry{
		TracerProvider:  tracerProvider,
		MeterProvider:   meterProvider,
		LoggerProvider:  loggerProvider,
		Tracer:          tracerProvider.Tracer(TracerName),
		Meter:           meterProvider.Meter(MeterName),
		UserCounter:     userCounter,
		RequestDuration: requestDuration,
	}
	return t, shutdown, nil
}

// RecordRequestDuration records the duration of an HTTP request
func (t *Telemetry) RecordRequestDuration(ctx context.Context, duration time.Duration, attrs ...attribute.KeyValue) {
	if t.RequestDuration != nil {
		t.RequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	}
}
