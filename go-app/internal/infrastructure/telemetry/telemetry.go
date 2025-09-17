package telemetry

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
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

type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
	Tracer         trace.Tracer
	Meter          metric.Meter
	UserCounter    metric.Int64Counter
	LogVerbosity   int
}

func Setup(ctx context.Context, cfg config.Config) (*Telemetry, func(context.Context) error, error) {
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
			semconv.ServiceNameKey.String(cfg.Otel.ServiceName),
			semconv.ServiceVersionKey.String(cfg.Otel.ServiceVersion),
			semconv.ServiceNamespaceKey.String(cfg.Otel.ServiceNamespace),
		),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return handleErr(fmt.Errorf("failed to create resource: %w", err))
	}

	protocol := cfg.Otel.Protocol
	if protocol == "" {
		protocol = "http"
	}
	slog.Info("Using OTLP protocol", "protocol", protocol, "endpoint", cfg.Otel.Endpoint)

	var (
		spanExporter sdktrace.SpanExporter
		metricReader sdkmetric.Reader
		logProcessor sdklog.Processor
	)

	// --- Exporter setup ---
	switch protocol {
	case "grpc":
		conn, err := grpc.NewClient(cfg.Otel.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			slog.Error("Failed to connect to OTLP gRPC", "endpoint", cfg.Otel.Endpoint, "err", err)
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
		logProcessor = newBatchProcessor(logExp, cfg.Otel)

	default: // HTTP
		traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Otel.Endpoint)}
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Otel.Endpoint)}
		logOpts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Otel.Endpoint)}

		// Add basic auth headers if credentials are provided
		if cfg.Otel.Username != "" && cfg.Otel.Password != "" {
			auth := cfg.Otel.Username + ":" + cfg.Otel.Password
			encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
			headers := map[string]string{
				"Authorization": "Basic " + encodedAuth,
			}
			traceOpts = append(traceOpts, otlptracehttp.WithHeaders(headers))
			metricOpts = append(metricOpts, otlpmetrichttp.WithHeaders(headers))
			logOpts = append(logOpts, otlploghttp.WithHeaders(headers))
		}

		if cfg.Otel.Insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
			logOpts = append(logOpts, otlploghttp.WithInsecure())
			slog.Warn("Using insecure HTTP connection", "endpoint", cfg.Otel.Endpoint)
		}

		spanExporter, err = otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			slog.Warn("OTLP trace exporter unreachable", "endpoint", cfg.Otel.Endpoint, "err", err)
			return handleErr(err)
		}

		metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			slog.Warn("OTLP metric exporter unreachable", "endpoint", cfg.Otel.Endpoint, "err", err)
			return handleErr(err)
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExp)

		logExp, err := otlploghttp.New(ctx, logOpts...)
		if err != nil {
			slog.Warn("OTLP log exporter unreachable", "endpoint", cfg.Otel.Endpoint, "err", err)
			return handleErr(err)
		}
		logProcessor = newBatchProcessor(logExp, cfg.Otel)
	}

	// --- Providers ---
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(spanExporter,
			sdktrace.WithMaxQueueSize(cfg.Otel.MaxQueueSize),
			sdktrace.WithBatchTimeout(time.Duration(cfg.Otel.BatchTimeoutSecs)*time.Second),
			sdktrace.WithExportTimeout(time.Duration(cfg.Otel.ExportTimeoutSecs)*time.Second)),
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

	// Configure slog based on configuration
	setupSlog(cfg.Otel, loggerProvider)

	// Create meter and instruments before starting runtime metrics
	meter := meterProvider.Meter(cfg.Otel.MeterName)
	userCounter, err := meter.Int64Counter("user_operations_total",
		metric.WithDescription("Counts user operations"),
		metric.WithUnit("{operation}"))
	if err != nil {
		return handleErr(fmt.Errorf("failed to create user counter: %w", err))
	}

	// Start runtime metrics collection
	if err := runtime.Start(runtime.WithMeterProvider(meterProvider)); err != nil {
		slog.Error("Failed to start runtime metrics", "err", err)
		return handleErr(err)
	}

	return &Telemetry{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		LoggerProvider: loggerProvider,
		Tracer:         tracerProvider.Tracer(cfg.Otel.TracerName),
		Meter:          meter,
		UserCounter:    userCounter,
		LogVerbosity:   cfg.Otel.LogVerbosity,
	}, shutdown, nil
}

func newBatchProcessor(exp sdklog.Exporter, cfg config.OtelConfig) sdklog.Processor {
	return sdklog.NewBatchProcessor(exp,
		sdklog.WithMaxQueueSize(cfg.MaxQueueSize),
		sdklog.WithExportInterval(time.Duration(cfg.ExportIntervalSecs)*time.Second),
		sdklog.WithExportTimeout(time.Duration(cfg.ExportTimeoutSecs)*time.Second),
	)
}

// setupSlog configures slog with stdout/stderr + OTEL output
func setupSlog(cfg config.OtelConfig, loggerProvider *sdklog.LoggerProvider) {
	var loggers []*slog.Logger

	// Add stdout/stderr logger if not OTEL-only
	if cfg.LogOutput != "otel" {
		output := os.Stdout
		if strings.ToLower(cfg.LogOutput) == "stderr" {
			output = os.Stderr
		}

		var handler slog.Handler
		if strings.ToLower(cfg.LogFormat) == "json" {
			handler = slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
		} else {
			handler = slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
		}
		loggers = append(loggers, slog.New(handler))
	}

	// Add OTEL logger
	loggers = append(loggers, otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider)))

	// Set default logger
	if len(loggers) == 1 {
		slog.SetDefault(loggers[0])
	} else {
		slog.SetDefault(slog.New(&multiHandler{loggers: loggers}))
	}
}

// multiHandler writes to multiple loggers
type multiHandler struct {
	loggers []*slog.Logger
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Return true if any of the underlying handlers would process this level
	for _, logger := range m.loggers {
		if logger.Handler().Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, logger := range m.loggers {
		logger.Handler().Handle(ctx, record)
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var newLoggers []*slog.Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, slog.New(logger.Handler().WithAttrs(attrs)))
	}
	return &multiHandler{loggers: newLoggers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	var newLoggers []*slog.Logger
	for _, logger := range m.loggers {
		newLoggers = append(newLoggers, slog.New(logger.Handler().WithGroup(name)))
	}
	return &multiHandler{loggers: newLoggers}
}
