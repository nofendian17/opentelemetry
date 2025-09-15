package worker

import (
	"context"

	"go-app/internal/infrastructure/kafka"
	"go-app/internal/infrastructure/telemetry"

	kgopkg "github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
)

// KafkaWorker handles Kafka message consumption and business logic processing
type KafkaWorker struct {
	consumer  *kafka.Consumer
	telemetry *telemetry.Telemetry
}

// NewKafkaWorker creates a new Kafka worker instance
func NewKafkaWorker(consumer *kafka.Consumer, tel *telemetry.Telemetry) *KafkaWorker {
	return &KafkaWorker{
		consumer:  consumer,
		telemetry: tel,
	}
}

// Start begins the Kafka consumer in a separate goroutine
func (w *KafkaWorker) Start(ctx context.Context) {
	go w.startConsumer(ctx)
}

// startConsumer starts the Kafka consumer with message handling
func (w *KafkaWorker) startConsumer(ctx context.Context) {
	messageHandler := func(ctx context.Context, record *kgopkg.Record) error {
		telemetry.Log(ctx, telemetry.LevelInfo, "Processing Kafka message", nil,
			attribute.String("kafka.topic", record.Topic),
			attribute.Int64("kafka.offset", record.Offset),
			attribute.String("kafka.value", string(record.Value)),
		)

		// Add your business logic processing here
		// For example, you could:
		// 1. Parse the message content
		// 2. Validate business rules
		// 3. Execute domain operations
		// 4. Update application state
		// 5. Trigger other business processes
		// 6. Send notifications or events

		return nil
	}

	if err := w.consumer.ConsumeWithTracing(ctx, messageHandler); err != nil {
		telemetry.Log(ctx, telemetry.LevelError, "Kafka consumer error", err)
	}
}
