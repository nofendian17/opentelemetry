package config

import (
	"os"
	"strconv"
)

// OtelConfig holds the configuration for OTel SDK
type OtelConfig struct {
	ServiceName      string
	ServiceVersion   string
	ServiceNamespace string
	Endpoint         string
	Insecure         bool
}

// GetEnv returns the value of an environment variable or a fallback value
func GetEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// GetEnvAsBool returns the value of an environment variable as a boolean
func GetEnvAsBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		b, err := strconv.ParseBool(val)
		if err == nil {
			return b
		}
	}
	return fallback
}

// LoadConfig loads configuration from environment variables
func LoadConfig() OtelConfig {
	return OtelConfig{
		ServiceName:      GetEnv("OTEL_SERVICE_NAME", "go-app"),
		ServiceVersion:   GetEnv("OTEL_SERVICE_VERSION", "v0.1.0"),
		ServiceNamespace: GetEnv("OTEL_SERVICE_NAMESPACE", ""),
		Endpoint:         GetEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		Insecure:         GetEnvAsBool("OTEL_EXPORTER_OTLP_INSECURE", true),
	}
}
