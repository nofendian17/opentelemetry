package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

// OtelConfig holds the configuration for OTel SDK
type OtelConfig struct {
	ServiceName      string
	ServiceVersion   string
	ServiceNamespace string
	Protocol         string
	Endpoint         string
	Insecure         bool
	AppPort          string
	// LogVerbosity controls the verbosity of logs (0 = minimal, 1 = standard, 2 = verbose)
	LogVerbosity int
	// TracerName is the name used for the tracer provider
	TracerName string
	// MeterName is the name used for the meter provider
	MeterName string
	// LogBodies controls whether request/response bodies are logged
	LogBodies bool
}

func LoadConfig() OtelConfig {
	// Set up viper to read from .env file
	viper.SetConfigFile(filepath.Join(".", ".env"))

	// Attempt to read the .env file
	if err := viper.ReadInConfig(); err != nil {
		// If we can't read the .env file, that's okay - we'll rely on environment variables
		// and defaults
	}

	// Enable reading configuration from environment variables
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("OTEL_SERVICE_NAME", "go-app")
	viper.SetDefault("OTEL_SERVICE_VERSION", "v0.1.0")
	viper.SetDefault("OTEL_SERVICE_NAMESPACE", "")
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	viper.SetDefault("OTEL_EXPORTER_OTLP_INSECURE", true)
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("OTEL_LOG_VERBOSITY", 1)
	viper.SetDefault("OTEL_TRACER_NAME", "go-app-tracer")
	viper.SetDefault("OTEL_METER_NAME", "go-app-meter")
	viper.SetDefault("DISABLE_BODY_LOGGING", false) // Default to logging bodies

	return OtelConfig{
		ServiceName:      viper.GetString("OTEL_SERVICE_NAME"),
		ServiceVersion:   viper.GetString("OTEL_SERVICE_VERSION"),
		ServiceNamespace: viper.GetString("OTEL_SERVICE_NAMESPACE"),
		Protocol:         viper.GetString("OTEL_EXPORTER_OTLP_PROTOCOL"),
		Endpoint:         viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:         viper.GetBool("OTEL_EXPORTER_OTLP_INSECURE"),
		AppPort:          viper.GetString("APP_PORT"),
		LogVerbosity:     viper.GetInt("OTEL_LOG_VERBOSITY"),
		TracerName:       viper.GetString("OTEL_TRACER_NAME"),
		MeterName:        viper.GetString("OTEL_METER_NAME"),
		LogBodies:        viper.GetBool("DISABLE_BODY_LOGGING"),
	}
}
