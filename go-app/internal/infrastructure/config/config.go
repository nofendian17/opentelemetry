package config

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

// OtelConfig holds the configuration for OTel SDK
type OtelConfig struct {
	ServiceName      string
	ServiceVersion   string
	ServiceNamespace string
	Endpoint         string
	Insecure         bool
	AppPort          string
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

func LoadConfig() OtelConfig {
	viper.AutomaticEnv()

	viper.SetDefault("OTEL_SERVICE_NAME", "go-app")
	viper.SetDefault("OTEL_SERVICE_VERSION", "v0.1.0")
	viper.SetDefault("OTEL_SERVICE_NAMESPACE", "")
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	viper.SetDefault("OTEL_EXPORTER_OTLP_INSECURE", true)
	viper.SetDefault("APP_PORT", "8080")

	return OtelConfig{
		ServiceName:      viper.GetString("OTEL_SERVICE_NAME"),
		ServiceVersion:   viper.GetString("OTEL_SERVICE_VERSION"),
		ServiceNamespace: viper.GetString("OTEL_SERVICE_NAMESPACE"),
		Endpoint:         viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Insecure:         viper.GetBool("OTEL_EXPORTER_OTLP_INSECURE"),
		AppPort:          viper.GetString("APP_PORT"),
	}
}
