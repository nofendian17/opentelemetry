package kafka

import (
	"context"
	"fmt"
	"time"

	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/telemetry"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Producer wraps kgo.Client for producing messages
type Producer struct {
	*kgo.Client
	tracer trace.Tracer
	tel    *telemetry.Telemetry
}

// Consumer wraps kgo.Client for consuming messages
type Consumer struct {
	*kgo.Client
	tracer trace.Tracer
	tel    *telemetry.Telemetry
}

// NewProducer creates a new Kafka producer with best practices configuration
func NewProducer(cfg config.KafkaConfig, tel *telemetry.Telemetry) (*Producer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.WithHooks(kotel.NewKotel().Hooks()...),
		kgo.ProducerBatchMaxBytes(1048576), // 1MB
		kgo.ProducerBatchCompression(kgo.GzipCompression()),
		kgo.ProducerLinger(5 * time.Millisecond),
		kgo.RequestTimeoutOverhead(10 * time.Second),
		kgo.ConnIdleTimeout(time.Duration(cfg.ConnIdleTime) * time.Second),
		kgo.DialTimeout(time.Duration(cfg.DialTimeout) * time.Second),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	telemetry.Log(context.Background(), telemetry.LevelInfo, "Successfully created Kafka producer", nil,
		attribute.StringSlice("kafka.brokers", cfg.Brokers),
		attribute.String("kafka.topic", cfg.Topic),
	)

	return &Producer{
		Client: client,
		tracer: tel.Tracer,
		tel:    tel,
	}, nil
}

// NewConsumer creates a new Kafka consumer with best practices configuration
func NewConsumer(cfg config.KafkaConfig, groupID string, tel *telemetry.Telemetry) (*Consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.WithHooks(kotel.NewKotel().Hooks()...),
		kgo.FetchMaxBytes(52428800), // 50MB
		kgo.FetchMinBytes(1),
		kgo.FetchMaxWait(500 * time.Millisecond),
		kgo.SessionTimeout(30 * time.Second),
		kgo.HeartbeatInterval(3 * time.Second),
		kgo.RebalanceTimeout(30 * time.Second),
		kgo.ConnIdleTimeout(time.Duration(cfg.ConnIdleTime) * time.Second),
		kgo.DialTimeout(time.Duration(cfg.DialTimeout) * time.Second),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	telemetry.Log(context.Background(), telemetry.LevelInfo, "Successfully created Kafka consumer", nil,
		attribute.StringSlice("kafka.brokers", cfg.Brokers),
		attribute.String("kafka.topic", cfg.Topic),
		attribute.String("kafka.consumer_group", groupID),
	)

	return &Consumer{
		Client: client,
		tracer: tel.Tracer,
		tel:    tel,
	}, nil
}

// ProduceWithTracing produces a message with tracing and error handling
func (p *Producer) ProduceWithTracing(ctx context.Context, topic string, key, value []byte) error {
	ctx, span := p.tracer.Start(ctx, "kafka.produce")
	defer span.End()

	span.SetAttributes(
		attribute.String("kafka.topic", topic),
		attribute.String("kafka.operation", "produce"),
		attribute.Int("kafka.message_size", len(value)),
	)

	record := &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: value,
	}

	// Produce asynchronously with callback
	results := p.ProduceSync(ctx, record)
	err := results.FirstErr()
	if err != nil {
		span.SetAttributes(attribute.Bool("kafka.error", true))
		return fmt.Errorf("failed to produce message: %w", err)
	}

	span.SetAttributes(attribute.Bool("kafka.success", true))
	telemetry.Log(ctx, telemetry.LevelInfo, "Message produced successfully", nil,
		attribute.String("kafka.topic", topic),
		attribute.Int("kafka.message_size", len(value)),
	)

	return nil
}

// ConsumeWithTracing consumes messages with tracing and error handling
func (c *Consumer) ConsumeWithTracing(ctx context.Context, handler func(ctx context.Context, record *kgo.Record) error) error {
	ctx, span := c.tracer.Start(ctx, "kafka.consume")
	defer span.End()

	for {
		select {
		case <-ctx.Done():
			telemetry.Log(ctx, telemetry.LevelInfo, "Kafka consumer shutting down", nil)
			return ctx.Err()
		default:
			fetches := c.PollFetches(ctx)

			if fetches.IsClientClosed() {
				telemetry.Log(ctx, telemetry.LevelInfo, "Kafka client closed, consumer stopping", nil)
				return nil
			}

			// Check for errors
			if errs := fetches.Errors(); len(errs) > 0 {
				continue
			}

			// Check if there are no records
			if fetches.NumRecords() == 0 {
				continue // No records, try again
			}

			var processedCount int
			fetches.EachRecord(func(record *kgo.Record) {
				recordCtx, recordSpan := c.tracer.Start(ctx, "kafka.process_record")
				recordSpan.SetAttributes(
					attribute.String("kafka.topic", record.Topic),
					attribute.Int64("kafka.offset", record.Offset),
					attribute.Int("kafka.partition", int(record.Partition)),
				)

				if err := handler(recordCtx, record); err != nil {
					recordSpan.SetAttributes(attribute.Bool("kafka.processing_error", true))
				} else {
					processedCount++
				}

				recordSpan.End()
			})

			if processedCount > 0 {
				telemetry.Log(ctx, telemetry.LevelInfo, "Processed Kafka messages", nil,
					attribute.Int("kafka.processed_count", processedCount),
				)
			}
		}
	}
}

// HealthCheck performs a health check on the Kafka connection
func (p *Producer) HealthCheck(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "kafka.health_check")
	defer span.End()

	// Simple ping by trying to get broker metadata
	results := make(chan error, 1)
	go func() {
		defer close(results)
		// Try a lightweight operation to check connectivity
		testRecord := &kgo.Record{
			Topic: "health-check-topic",
			Key:   []byte("health-check"),
			Value: []byte("ping"),
		}

		produceResults := p.ProduceSync(ctx, testRecord)
		results <- produceResults.FirstErr()
	}()

	select {
	case err := <-results:
		if err != nil {
			span.SetAttributes(attribute.Bool("kafka.healthy", false))
			return fmt.Errorf("kafka health check failed: %w", err)
		}
	case <-time.After(5 * time.Second):
		span.SetAttributes(attribute.Bool("kafka.healthy", false))
		return fmt.Errorf("kafka health check timed out")
	}

	span.SetAttributes(attribute.Bool("kafka.healthy", true))
	return nil
}

// Close closes the Kafka client
func (p *Producer) Close() {
	p.Client.Close()
}

// Close closes the Kafka client
func (c *Consumer) Close() {
	c.Client.Close()
}
