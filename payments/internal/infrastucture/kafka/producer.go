package kafka_infra

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer interface {
	Produce(ctx context.Context, topic string, message []byte) error
	Close() error
}

type KafkaProducer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewProducer(brokers []string, _ string, logger *zap.Logger) *KafkaProducer {
	// The topic parameter is no longer used to configure the writer directly,
	// as the Produce method now accepts a topic argument.
	// We keep it in the signature for now if other parts of the system expect it,
	// but mark it as unused (underscore) to indicate it's not used in this function body.
	// Alternatively, if no other part relies on this signature, it could be removed.
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		// Topic: topic, // Topic is now specified per message in Produce method
		Balancer: &kafka.LeastBytes{},
	})
	return &KafkaProducer{writer: writer, logger: logger}
}

func (p *KafkaProducer) Produce(ctx context.Context, topic string, message []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Value: message,
	}
	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Error("Failed to produce message to Kafka topic",
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("failed to produce message: %w", err)
	}
	p.logger.Debug("Produced message to topic", zap.String("topic", topic))
	return nil
}

func (p *KafkaProducer) Close() error {
	if err := p.writer.Close(); err != nil {
		p.logger.Error("Failed to close Kafka producer", zap.Error(err))
		return fmt.Errorf("failed to close Kafka producer: %w", err)
	}
	p.logger.Info("Kafka producer closed.")
	return nil
}
