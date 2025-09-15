package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	Otel     OtelConfig
	Kafka    KafkaConfig
	Redis    RedisConfig
	Postgres PostgresConfig
}

// OtelConfig holds the configuration for OTel SDK
type OtelConfig struct {
	ServiceName        string
	ServiceVersion     string
	ServiceNamespace   string
	Protocol           string
	Endpoint           string
	Insecure           bool
	AppPort            string
	LogVerbosity       int
	TracerName         string
	MeterName          string
	LogBodies          bool
	ExportIntervalSecs int
	ExportTimeoutSecs  int
	MaxQueueSize       int
	BatchTimeoutSecs   int
	LogOutput          string // "stdout", "stderr", "otel"
	LogFormat          string // "text", "json"
}

// KafkaConfig holds the configuration for Kafka
type KafkaConfig struct {
	Brokers       []string
	Topic         string
	ConsumerGroup string
	BatchSize     int
	DialTimeout   int // seconds
	ConnIdleTime  int // seconds
}

// RedisConfig holds the configuration for Redis
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	MaxRetries   int
	DialTimeout  int // seconds
	ReadTimeout  int // seconds
	WriteTimeout int // seconds
	PoolSize     int
	MinIdleConns int
	MaxConnAge   int // minutes
	PoolTimeout  int // seconds
	IdleTimeout  int // minutes
}

// PostgresConfig holds the configuration for PostgreSQL
type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // minutes
	ConnMaxIdleTime int // minutes
}

func LoadConfig() Config {
	// Set up viper to read from .env file
	viper.SetConfigFile(filepath.Join(".", ".env"))

	// Attempt to read the .env file
	if err := viper.ReadInConfig(); err != nil {
		// If we can't read the .env file, that's okay - we'll rely on environment variables
		// and defaults
	}

	// Enable reading configuration from environment variables
	viper.AutomaticEnv()

	// Set defaults for OTel
	viper.SetDefault("OTEL_SERVICE_NAME", "go-app")
	viper.SetDefault("OTEL_SERVICE_VERSION", "v0.1.0")
	viper.SetDefault("OTEL_SERVICE_NAMESPACE", "")
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318")
	viper.SetDefault("OTEL_EXPORTER_OTLP_INSECURE", true)
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("OTEL_LOG_VERBOSITY", 1)
	viper.SetDefault("OTEL_TRACER_NAME", "go-app-tracer")
	viper.SetDefault("OTEL_METER_NAME", "go-app-meter")
	viper.SetDefault("DISABLE_BODY_LOGGING", false)
	viper.SetDefault("OTEL_EXPORT_INTERVAL_SECS", 60)
	viper.SetDefault("OTEL_EXPORT_TIMEOUT_SECS", 30)
	viper.SetDefault("OTEL_MAX_QUEUE_SIZE", 10000)
	viper.SetDefault("OTEL_BATCH_TIMEOUT_SECS", 5)
	viper.SetDefault("OTEL_LOG_OUTPUT", "stdout")
	viper.SetDefault("OTEL_LOG_FORMAT", "text")

	// Set defaults for Kafka
	viper.SetDefault("KAFKA_BROKERS", "localhost:9092")
	viper.SetDefault("KAFKA_TOPIC", "go-app-events")
	viper.SetDefault("KAFKA_CONSUMER_GROUP", "go-app-consumer-group")
	viper.SetDefault("KAFKA_BATCH_SIZE", 100)
	viper.SetDefault("KAFKA_DIAL_TIMEOUT", 15)
	viper.SetDefault("KAFKA_CONN_IDLE_TIME", 20)

	// Set defaults for Redis
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_MAX_RETRIES", 3)
	viper.SetDefault("REDIS_DIAL_TIMEOUT", 5)
	viper.SetDefault("REDIS_READ_TIMEOUT", 3)
	viper.SetDefault("REDIS_WRITE_TIMEOUT", 3)
	viper.SetDefault("REDIS_POOL_SIZE", 10)
	viper.SetDefault("REDIS_MIN_IDLE_CONNS", 2)
	viper.SetDefault("REDIS_MAX_CONN_AGE", 30)
	viper.SetDefault("REDIS_POOL_TIMEOUT", 4)
	viper.SetDefault("REDIS_IDLE_TIMEOUT", 5)

	// Set defaults for Postgres
	viper.SetDefault("POSTGRES_DSN", "postgres://user:password@localhost:5432/go-app?sslmode=disable")
	viper.SetDefault("POSTGRES_MAX_OPEN_CONNS", 25)
	viper.SetDefault("POSTGRES_MAX_IDLE_CONNS", 10)
	viper.SetDefault("POSTGRES_CONN_MAX_LIFETIME", 5)
	viper.SetDefault("POSTGRES_CONN_MAX_IDLE_TIME", 5)

	return Config{
		Otel: OtelConfig{
			ServiceName:        viper.GetString("OTEL_SERVICE_NAME"),
			ServiceVersion:     viper.GetString("OTEL_SERVICE_VERSION"),
			ServiceNamespace:   viper.GetString("OTEL_SERVICE_NAMESPACE"),
			Protocol:           viper.GetString("OTEL_EXPORTER_OTLP_PROTOCOL"),
			Endpoint:           viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
			Insecure:           viper.GetBool("OTEL_EXPORTER_OTLP_INSECURE"),
			AppPort:            viper.GetString("APP_PORT"),
			LogVerbosity:       viper.GetInt("OTEL_LOG_VERBOSITY"),
			TracerName:         viper.GetString("OTEL_TRACER_NAME"),
			MeterName:          viper.GetString("OTEL_METER_NAME"),
			LogBodies:          !viper.GetBool("DISABLE_BODY_LOGGING"),
			ExportIntervalSecs: viper.GetInt("OTEL_EXPORT_INTERVAL_SECS"),
			ExportTimeoutSecs:  viper.GetInt("OTEL_EXPORT_TIMEOUT_SECS"),
			MaxQueueSize:       viper.GetInt("OTEL_MAX_QUEUE_SIZE"),
			BatchTimeoutSecs:   viper.GetInt("OTEL_BATCH_TIMEOUT_SECS"),
			LogOutput:          viper.GetString("OTEL_LOG_OUTPUT"),
			LogFormat:          viper.GetString("OTEL_LOG_FORMAT"),
		},
		Kafka: KafkaConfig{
			Brokers:       viper.GetStringSlice("KAFKA_BROKERS"),
			Topic:         viper.GetString("KAFKA_TOPIC"),
			ConsumerGroup: viper.GetString("KAFKA_CONSUMER_GROUP"),
			BatchSize:     viper.GetInt("KAFKA_BATCH_SIZE"),
			DialTimeout:   viper.GetInt("KAFKA_DIAL_TIMEOUT"),
			ConnIdleTime:  viper.GetInt("KAFKA_CONN_IDLE_TIME"),
		},
		Redis: RedisConfig{
			Addr:         viper.GetString("REDIS_ADDR"),
			Password:     viper.GetString("REDIS_PASSWORD"),
			DB:           viper.GetInt("REDIS_DB"),
			MaxRetries:   viper.GetInt("REDIS_MAX_RETRIES"),
			DialTimeout:  viper.GetInt("REDIS_DIAL_TIMEOUT"),
			ReadTimeout:  viper.GetInt("REDIS_READ_TIMEOUT"),
			WriteTimeout: viper.GetInt("REDIS_WRITE_TIMEOUT"),
			PoolSize:     viper.GetInt("REDIS_POOL_SIZE"),
			MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
			MaxConnAge:   viper.GetInt("REDIS_MAX_CONN_AGE"),
			PoolTimeout:  viper.GetInt("REDIS_POOL_TIMEOUT"),
			IdleTimeout:  viper.GetInt("REDIS_IDLE_TIMEOUT"),
		},
		Postgres: PostgresConfig{
			DSN:             viper.GetString("POSTGRES_DSN"),
			MaxOpenConns:    viper.GetInt("POSTGRES_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("POSTGRES_MAX_IDLE_CONNS"),
			ConnMaxLifetime: viper.GetInt("POSTGRES_CONN_MAX_LIFETIME"),
			ConnMaxIdleTime: viper.GetInt("POSTGRES_CONN_MAX_IDLE_TIME"),
		},
	}
}
