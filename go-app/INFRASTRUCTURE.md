# Infrastructure Implementation with Best Practices

This document provides comprehensive documentation for the Kafka, Redis, and Postgres infrastructure implementation with best practices for clean architecture, maintainability, and scalability.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Infrastructure Components](#infrastructure-components)
4. [Configuration Management](#configuration-management)
5. [Best Practices Implemented](#best-practices-implemented)
6. [Local Development Setup](#local-development-setup)
7. [Usage Examples](#usage-examples)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Production Considerations](#production-considerations)
10. [Troubleshooting](#troubleshooting)

## Overview

The infrastructure implementation follows clean architecture principles with:

- **Separation of Concerns**: Each infrastructure component is isolated with clear interfaces
- **Dependency Inversion**: Business logic depends on abstractions, not implementations
- **Configuration Management**: Environment-based configuration with sensible defaults
- **Observability**: Built-in logging, tracing, and metrics for all operations
- **Connection Pooling**: Optimized connection management for all services
- **Health Checks**: Comprehensive health monitoring for each component

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
├─────────────────────────────────────────────────────────────┤
│                 Infrastructure Layer                        │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Redis     │  │   Kafka     │  │     PostgreSQL      │  │
│  │             │  │             │  │                     │  │
│  │ • Caching   │  │ • Messaging │  │ • Primary Storage   │  │
│  │ • Sessions  │  │ • Events    │  │ • Transactions     │  │
│  │ • Rate Lmt  │  │ • Streaming │  │ • ACID Properties   │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │              Observability Layer                       │  │
│  │                                                         │  │
│  │  • OpenTelemetry Tracing                               │  │
│  │  • Structured Logging                                  │  │
│  │  • Metrics Collection                                  │  │
│  │  • Health Checks                                       │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Infrastructure Components

### Redis

**Purpose**: High-performance caching, session storage, and rate limiting

**Features Implemented**:
- Connection pooling with configurable parameters
- Retry logic with exponential backoff
- Circuit breaker pattern for fault tolerance
- Comprehensive health checks
- OpenTelemetry tracing for all operations
- Structured logging with context

**Configuration Options**:
```env
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_MAX_RETRIES=3
REDIS_DIAL_TIMEOUT=5
REDIS_READ_TIMEOUT=3
REDIS_WRITE_TIMEOUT=3
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=2
REDIS_MAX_CONN_AGE=30
REDIS_POOL_TIMEOUT=4
REDIS_IDLE_TIMEOUT=5
```

### Kafka

**Purpose**: Event streaming, message queuing, and real-time data processing

**Features Implemented**:
- Producer and consumer wrappers with best practices
- Configurable batch sizes and compression
- Error handling with retry mechanisms
- Dead letter queue support
- OpenTelemetry tracing for message lifecycle
- Graceful shutdown handling

**Configuration Options**:
```env
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=go-app-events
KAFKA_CONSUMER_GROUP=go-app-consumer-group
KAFKA_BATCH_SIZE=100
KAFKA_DIAL_TIMEOUT=15
KAFKA_CONN_IDLE_TIME=20
```

### PostgreSQL

**Purpose**: Primary data storage with ACID properties

**Features Implemented**:
- Connection pooling with lifecycle management
- Prepared statement optimization
- Transaction support with context
- Query tracing and performance monitoring
- Connection health monitoring
- Automatic connection recovery

**Configuration Options**:
```env
POSTGRES_DSN=postgres://user:password@localhost:5432/go-app?sslmode=disable
POSTGRES_MAX_OPEN_CONNS=25
POSTGRES_MAX_IDLE_CONNS=10
POSTGRES_CONN_MAX_LIFETIME=5
POSTGRES_CONN_MAX_IDLE_TIME=5
```

## Configuration Management

### Environment Variables

All infrastructure components support environment-based configuration with sensible defaults:

```bash
# Core application settings
OTEL_SERVICE_NAME=go-app
OTEL_SERVICE_VERSION=v1.0.0
APP_PORT=8080

# Database configuration
POSTGRES_DSN=postgres://go_app_user:go_app_password@localhost:5432/go_app?sslmode=disable
POSTGRES_MAX_OPEN_CONNS=25
POSTGRES_MAX_IDLE_CONNS=10

# Cache configuration
REDIS_ADDR=localhost:6379
REDIS_POOL_SIZE=10
REDIS_MAX_RETRIES=3

# Message queue configuration
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=go-app-events
KAFKA_CONSUMER_GROUP=go-app-consumer-group
```

### Configuration Validation

The application validates all configuration on startup and provides detailed error messages for invalid settings.

## Best Practices Implemented

### 1. Connection Pooling

**Redis**:
- Configurable pool size and idle connections
- Connection reuse and lifecycle management
- Automatic connection recovery

**PostgreSQL**:
- Optimized pool parameters for different workloads
- Connection lifetime management
- Prepared statement caching

**Kafka**:
- Producer and consumer connection pooling
- Batch processing for efficiency
- Connection idle timeout management

### 2. Error Handling

```go
// Structured error handling with context
type DomainError struct {
    Code    ErrorCode
    Message string
    Context map[string]interface{}
    Cause   error
}

// Retry logic with exponential backoff
func retryWithBackoff(operation func() error, maxRetries int) error {
    // Implementation with jitter and exponential backoff
}
```

### 3. Observability

**Distributed Tracing**:
- Every database query, cache operation, and message is traced
- Correlation IDs for request tracking
- Performance metrics collection

**Structured Logging**:
- Contextual logging with attributes
- Error correlation and debugging
- Performance monitoring

**Health Checks**:
- Deep health checks for all components
- Dependency health monitoring
- Graceful degradation support

### 4. Security

**Database Security**:
- Connection string sanitization for logs
- Prepared statements to prevent SQL injection
- Connection encryption support

**Redis Security**:
- Optional password authentication
- Command sanitization for logs
- Network security considerations

## Local Development Setup

### 1. Start All Services (Infrastructure + Application)

```bash
# Start all services including the go-app
docker compose up -d

# Check service health
docker compose ps
```

### 2. Verify Services

```bash
# Test PostgreSQL connection
docker exec -it go-app-postgres psql -U go_app_user -d go_app -c "SELECT version();"

# Test Redis connection
docker exec -it go-app-redis redis-cli ping

# Check Kafka topics
docker exec -it go-app-kafka kafka-topics --bootstrap-server localhost:9092 --list
```

### 3. Access Management Interfaces

- **Go Application**: http://localhost:8080
- **Adminer (PostgreSQL)**: http://localhost:8081
- **Kafka UI**: http://localhost:8090
- **MinIO Console**: http://localhost:9001

### 4. Environment Configuration

Create a `.env` file in the go-app directory:

```bash
# Database
POSTGRES_DSN=postgres://go_app_user:go_app_password@localhost:5432/go_app?sslmode=disable

# Cache
REDIS_ADDR=localhost:6379

# Message Queue
KAFKA_BROKERS=localhost:9092

# Application
APP_PORT=8080
OTEL_SERVICE_NAME=go-app
```

## Usage Examples

### Redis Operations

```go
// Set with expiration and tracing
err := redisClient.SetWithTracing(ctx, "user:123", userData, 1*time.Hour)

// Get with error handling
value, err := redisClient.GetWithTracing(ctx, "user:123")
if err == redis.Nil {
    // Key not found
} else if err != nil {
    // Handle error
}

// Delete multiple keys
err := redisClient.DelWithTracing(ctx, "user:123", "session:456")
```

### Kafka Operations

```go
// Produce message with tracing
err := producer.ProduceWithTracing(ctx, "user-events", 
    []byte("user:123"), 
    []byte(`{"event": "user_created", "id": 123}`))

// Consume messages with handler
messageHandler := func(ctx context.Context, record *kgo.Record) error {
    // Process message
    log.Printf("Received: %s", string(record.Value))
    return nil
}

err := consumer.ConsumeWithTracing(ctx, messageHandler)
```

### PostgreSQL Operations

```go
// Execute query with tracing
result, err := pgClient.ExecWithTracing(ctx, 
    "INSERT INTO users (name, email) VALUES ($1, $2)",
    "John Doe", "john@example.com")

// Query with tracing
rows, err := pgClient.QueryWithTracing(ctx,
    "SELECT id, name, email FROM users WHERE id = $1", userID)

// Transaction with tracing
tx, err := pgClient.BeginTxWithTracing(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

// ... perform operations ...

err = tx.Commit()
```

## Monitoring and Observability

### Health Check Endpoints

The application provides health check endpoints for monitoring:

```bash
# Application health
curl http://localhost:8080/health

# Individual service health
curl http://localhost:8080/health/postgres
curl http://localhost:8080/health/redis
curl http://localhost:8080/health/kafka
```

### Metrics Collection

Key metrics collected:

- **Connection Pool Stats**: Active, idle, and waiting connections
- **Query Performance**: Query duration, success/failure rates
- **Cache Performance**: Hit/miss ratios, operation latency
- **Message Processing**: Throughput, latency, error rates

### Tracing

All operations are traced with OpenTelemetry:

- Database queries with SQL statements
- Cache operations with keys and values
- Message production and consumption
- HTTP requests and responses

## Production Considerations

### 1. Security

```bash
# Use secure connection strings
POSTGRES_DSN=postgres://user:password@host:5432/db?sslmode=require

# Enable Redis authentication
REDIS_PASSWORD=secure_password

# Configure Kafka security
KAFKA_SECURITY_PROTOCOL=SASL_SSL
```

### 2. Performance Tuning

**PostgreSQL**:
- Adjust connection pool sizes based on workload
- Configure statement timeout
- Enable connection pooling at application level

**Redis**:
- Monitor memory usage and configure eviction policies
- Use Redis Cluster for high availability
- Configure persistence based on durability requirements

**Kafka**:
- Tune batch sizes for throughput vs latency
- Configure replication factor for durability
- Monitor consumer lag

### 3. High Availability

- Use PostgreSQL streaming replication
- Deploy Redis in cluster mode
- Configure Kafka with multiple brokers
- Implement circuit breakers for fault tolerance

### 4. Monitoring

- Set up alerts for connection pool exhaustion
- Monitor query performance and slow queries
- Track cache hit rates and memory usage
- Monitor Kafka consumer lag

## Troubleshooting

### Common Issues

1. **Connection Pool Exhaustion**:
   - Increase max connections or decrease connection lifetime
   - Check for connection leaks in application code
   - Monitor connection usage patterns

2. **Redis Memory Issues**:
   - Configure maxmemory policy
   - Monitor key expiration settings
   - Check for large keys or memory leaks

3. **Kafka Consumer Lag**:
   - Increase consumer instances
   - Optimize message processing logic
   - Check for slow downstream dependencies

4. **Database Performance**:
   - Analyze slow queries
   - Check index usage
   - Monitor connection pool stats

### Debugging

Enable debug logging:

```bash
OTEL_LOG_VERBOSITY=2
```

Check service logs:

```bash
# Application logs
docker compose logs go-app

# Infrastructure service logs
docker compose logs postgres
docker compose logs redis
docker compose logs kafka

# All services
docker compose logs
```

### Health Check Failures

If health checks fail:

1. Verify service connectivity
2. Check authentication credentials
3. Monitor resource usage (CPU, memory, disk)
4. Review service-specific logs

## Conclusion

This infrastructure implementation provides a solid foundation for building scalable, maintainable applications with:

- **Clean Architecture**: Clear separation of concerns and dependency inversion
- **Best Practices**: Connection pooling, error handling, and observability
- **Production Ready**: Security, monitoring, and high availability considerations
- **Developer Friendly**: Comprehensive documentation and local development setup

The implementation follows Go idioms and industry best practices, making it suitable for both small applications and large-scale enterprise systems.
